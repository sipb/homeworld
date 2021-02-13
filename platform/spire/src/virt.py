import atexit
import concurrent.futures
import contextlib
import hashlib
import os
import subprocess
import tempfile
import threading
import time

import access
import command
import configuration
import infra
import iso
import seq
import util
import verify


def get_bridge(ip):
    """The name to be used for the bridge interface connecting the VMs and the host."""
    return "spirebr%s" % ip.packed.hex().upper()


def get_bridge_access(ip):
    """The name to be used for the vlan interface providing host-side access to the VMs."""
    return "spireac%s" % ip.packed.hex().upper()


def get_node_tap(node):
    """The name to be used for the tap interface providing network to a particular QEMU VM."""
    # maximum length: 15 characters
    return "spirtap%s" % node.ip.packed.hex().upper()


def determine_topology():
    config = configuration.get_config()
    gateway_ip = next(config.cidr_nodes.hosts())
    gateway = "%s/%d" % (gateway_ip, config.cidr_nodes.prefixlen)
    taps = []
    hosts = {}
    for node in config.nodes:
        if node.ip not in config.cidr_nodes:
            command.fail("invalid topology: address %s is not in CIDR %s" % (node.ip, config.cidr_nodes))
        taps.append(get_node_tap(node))
        hosts["%s.%s" % (node.hostname, config.external_domain)] = node.ip
    bridge_name = get_bridge(gateway_ip)
    if config.vlan != 0:
        access_name = get_bridge_access(gateway_ip)
    else:
        access_name = bridge_name
    return gateway, taps, bridge_name, access_name, hosts, config.vlan


def sudo(*command):
    subprocess.check_call(["sudo"] + list(command))


def sudo_ok(*command):
    return subprocess.call(["sudo"] + list(command)) == 0


def sysctl_set(key, value):
    sudo("sysctl", "-w", "--", "%s=%s" % (key, value))


def bridge_up(bridge_name, access_name, address, vlan):
    sudo("brctl", "addbr", bridge_name)
    sudo("ip", "link", "set", bridge_name, "up")
    if access_name != bridge_name:
        sudo("ip", "link", "add", "link", bridge_name, "name", access_name, "type", "vlan", "id", str(vlan))
        sudo("ip", "link", "set", access_name, "up")
    sudo("ip", "addr", "add", address, "dev", access_name)


def bridge_down(bridge_name, access_name, address):
    ok = sudo_ok("ip", "addr", "del", address, "dev", access_name)
    if access_name != bridge_name:
        ok &= sudo_ok("ip", "link", "set", access_name, "down")
        ok &= sudo_ok("ip", "link", "del", access_name)
    ok &= sudo_ok("ip", "link", "set", bridge_name, "down")
    ok &= sudo_ok("brctl", "delbr", bridge_name)
    return ok


def tap_up(bridge_name, tap):
    sudo("ip", "tuntap", "add", "user", os.getenv("USER"), "mode", "tap", tap)
    sudo("ip", "link", "set", tap, "up", "promisc", "on")
    sudo("brctl", "addif", bridge_name, tap)


def tap_down(bridge_name, tap):
    ok = sudo_ok("brctl", "delif", bridge_name, tap)
    ok &= sudo_ok("ip", "link", "set", tap, "down")
    ok &= sudo_ok("ip", "tuntap", "del", "mode", "tap", tap)
    return ok


def does_link_exist(link):
    return subprocess.check_call(["ip", "link", "show", "dev", link],
                                 stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL) == 0


def get_upstream_link():
    lines = subprocess.check_output(["ip", "-o", "-d", "route"]).decode().split("\n")
    defaults = [line for line in lines if line.startswith("unicast default via")]
    if len(defaults) != 1:
        command.fail("cannot determine upstream link from ip route output")
    link = defaults[0].split(" dev ")[1].split(" ")[0]
    if not does_link_exist(link):
        command.fail("failed to correctly determine upstream link: '%s' does not exist" % link)
    return link


def routing_up(access_name, upstream_link):
    sudo("iptables", "-I", "INPUT", "1", "-i", access_name, "-j", "ACCEPT")
    sudo("iptables", "-I", "FORWARD", "1", "-i", access_name, "-o", upstream_link, "-j", "ACCEPT")
    sudo("iptables", "-I", "FORWARD", "1", "-i", upstream_link, "-o", access_name, "-j", "ACCEPT")
    sudo("iptables", "-I", "FORWARD", "1", "-i", access_name, "-o", access_name, "-j", "ACCEPT")
    sudo("iptables", "-t", "nat", "-I", "POSTROUTING", "1", "-o", upstream_link, "-j", "MASQUERADE")


def routing_down(access_name, upstream_link):
    ok = sudo_ok("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", upstream_link, "-j", "MASQUERADE")
    ok &= sudo_ok("iptables", "-D", "FORWARD", "-i", access_name, "-o", access_name, "-j", "ACCEPT")
    ok &= sudo_ok("iptables", "-D", "FORWARD", "-i", upstream_link, "-o", access_name, "-j", "ACCEPT")
    ok &= sudo_ok("iptables", "-D", "FORWARD", "-i", access_name, "-o", upstream_link, "-j", "ACCEPT")
    ok &= sudo_ok("iptables", "-D", "INPUT", "-i", access_name, "-j", "ACCEPT")
    return ok


def sudo_update_file_by_filter(filename, discard_predicate):
    with tempfile.NamedTemporaryFile(mode="w") as fw:
        with open(filename, "r") as fr:
            for line in fr:
                line = line.rstrip("\n")
                if not discard_predicate(line):
                    fw.write(line + "\n")
        fw.flush()
        sudo("cp", "--", fw.name, filename)


def sudo_append_to_file(filename, lines):
    subprocess.run(["sudo", "tee", "-a", "--", filename], stdout=subprocess.DEVNULL, check=True,
                          input="".join(("%s\n" % line) for line in lines).encode())


def hosts_up(hosts):
    for host, ip in hosts.items():
        if "\t" in host:
            command.fail("expected no tabs in hostname %s" % repr(host))
        assert "\t" not in str(ip)
    sudo_append_to_file("/etc/hosts", ["%s\t%s" % (ip, hostname) for hostname, ip in hosts.items()])


def hosts_down(hosts):
    def is_our_host(line):
        if line.count("\t") != 1:
            return False
        ip, hostname = line.split("\t")
        return hostname in hosts and str(hosts[hostname]) == ip
    sudo_update_file_by_filter("/etc/hosts", discard_predicate=is_our_host)


def net_up_inner(gateway_ip, taps, bridge_name, access_name, hosts, vlan):
    upstream_link = get_upstream_link()

    sysctl_set("net.ipv4.ip_forward", 1)

    try:
        bridge_up(bridge_name, access_name, gateway_ip, vlan)
        for tap in taps:
            tap_up(bridge_name, tap)

        routing_up(access_name, upstream_link)

        hosts_up(hosts)
    except Exception as e:
        print("woops, tearing down...")
        if not net_down():
            print("could not tear down")
        raise e


def net_down_inner(gateway_ip, taps, bridge_name, access_name, hosts, fail=False):
    upstream_link = get_upstream_link()

    hosts_down(hosts)

    ok = routing_down(access_name, upstream_link)

    for tap in taps:
        ok &= tap_down(bridge_name, tap)

    ok &= bridge_down(bridge_name, access_name, gateway_ip)

    if not ok and fail:
        command.fail("tearing down network failed (maybe it was already torn down?)")
    return ok


@command.wrap
def net_up():
    "bring up local testing network"

    gateway_ip, taps, bridge_name, access_name, hosts, vlan = determine_topology()
    net_up_inner(gateway_ip, taps, bridge_name, access_name, hosts, vlan)


@command.wrap
def net_down(fail: bool=False):
    """
    bring down local testing network

    fail: raise an exception if bringing down the network fails; in
    particular this occurs if it was already down.
    """
    gateway_ip, taps, bridge_name, access_name, hosts, vlan = determine_topology()
    return net_down_inner(gateway_ip, taps, bridge_name, access_name, hosts, fail)


@contextlib.contextmanager
def net_context():
    gateway_ip, taps, bridge_name, access_name, hosts, vlan = determine_topology()
    net_up_inner(gateway_ip, taps, bridge_name, access_name, hosts, vlan)
    try:
        yield
    finally:
        net_down_inner(gateway_ip, taps, bridge_name, access_name, hosts, fail=True)


def modprobe(*modules):
    loaded = [l.split()[0] for l in subprocess.check_output(["lsmod"]).decode().split("\n") if l.strip()]
    needed = [mod for mod in modules if mod not in loaded]
    if needed:
        print("loading modules", *needed)
        sudo("modprobe", "-a", "--", *needed)
    else:
        print("modules already loaded:", *modules)


class TerminationContext:
    def __init__(self):
        self._sessions_lock = threading.Lock()
        self._sessions = []

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.kill_all_sessions()

    @property
    def sessions(self):
        with self._sessions_lock:
            return list(self._sessions)

    def add_session(self, s: "QEMUSession"):
        with self._sessions_lock:
            if s not in self._sessions:
                self._sessions.append(s)

    def remove_session(self, s: "QEMUSession"):
        with self._sessions_lock:
            if s in self._sessions:
                self._sessions.remove(s)

    def kill_all_sessions(self):
        for session in self.sessions:
            session.kill()
            session.wait()


class DebugContext:
    def __init__(self, persistent=False):
        self.persistent = persistent

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        if self.persistent or exc_type is not None:
            self.debug_shell()

    def debug_shell(self):
        print("launching debug shell...")
        subprocess.call(["bash"])
        print("debug shell quit")


class VirtualMachine:
    def __init__(self, node, tc: TerminationContext, cd=None, cpus=12, memory=2500, cdrom_install=False, debug_qemu=False):
        self.node = node
        self.cd = cd
        self.cpus = cpus
        self.memory = memory
        self.cdrom_install = cdrom_install
        self.netif = get_node_tap(node)
        self.tc = tc
        self.debug_qemu = debug_qemu

    @property
    def hostname(self):
        return self.node.hostname

    @property
    def hd(self):
        return os.path.join(configuration.get_project(), "virt-local", "disk-%s.qcow2" % self.hostname)

    @property
    def macaddress(self):
        if self.netif is None:
            return None
        digest = hashlib.sha256(self.netif.encode()).hexdigest()[-6:]
        return "52:54:00:%s:%s:%s" % (digest[0:2], digest[2:4], digest[4:6])

    def generate_qemu_args(self):
        args = ["qemu-system-x86_64"]
        args += ["-nographic", "-serial", "mon:stdio"]
        args += ["-machine", "accel=kvm", "-cpu", "host"]
        if self.cd is None:
            args += ["-drive", "media=disk,file=" + self.hd]
            args += ["-boot", "c"]
        else:
            # snapshot means that the installer drive won't be modifiable by the VMs, which is useful in qemu 3.0.0+ to
            # avoid locking problems when more than one VM is running at the same time.
            args += ["-drive", "media=%s,format=raw,file=%s,snapshot=on" % ("cdrom" if self.cdrom_install else "disk", self.cd)]
            args += ["-drive", "media=disk,file=" + self.hd]
            args += ["-boot", "c"]
        args += ["-no-reboot"]
        args += ["-smp", "%d" % int(self.cpus), "-m", "%d" % int(self.memory)]
        if self.netif is None:
            args += ["-net", "none"]
        else:
            args += ["-net", "nic,macaddr=%s" % self.macaddress]
            args += ["-net", "tap,ifname=%s,script=no,downscript=no" % self.netif]
        return args

    def check_module_loaded(self):
        # TODO: don't blindly load kvm_intel; check type of system first
        modprobe("kvm", "kvm_intel")

    def boot_with_io(self, phase, output_callback=None, text: bytes=None, delay=None, chardelay=None):
        self.check_module_loaded()

        log_output = open("log.%s.%s" % (self.node.hostname, phase), "wb")
        try:
            if output_callback:
                output = SplitWriter(log_output, LineCollector(output_callback))
            else:
                output = log_output

            if text:
                assert delay is not None and chardelay is not None
                stdin = delayed_yielder(delay, chardelay, text)
            else:
                stdin = None

            return QEMUSession(self, stdin, output)
        except BaseException as e:
            log_output.close()
            raise e

    def wait_and_pull_supervisor_key(self, fingerprints):
        last_error = None
        for i in range(60):
            time.sleep(1)
            try:
                # needs to be insecure because we aren't ready to validate the new hostkey yet
                verify.check_supervisor_accessible(insecure=True)
                break
            except Exception as e:
                print("[delayed supervisor key pull due to error]")
                last_error = e
        else:
            print("[supervisor key pull timeout reached; key pull will likely fail]")
            print("[last error: %s]" % last_error)
        access.pull_supervisor_key(fingerprints)

    def boot_launch(self, autoadd_fingerprint=False):
        if not autoadd_fingerprint:
            return self.boot_with_io("launch")
        else:
            extractor = FingerprintExtractor(self.wait_and_pull_supervisor_key)
            self.boot_with_io("launch", extractor.process_line)
            extractor.wait()

    def boot_install_supervisor(self):
        self.boot_install("manual")

    def boot_install_and_admit(self):
        bootstrap_token = infra.admit(self.hostname)
        if any(c.isspace() for c in bootstrap_token):
            raise Exception("expected no spaces in bootstrap token")
        self.boot_install(bootstrap_token)

    def boot_install(self, bootstrap_token):
        self.create_disk()
        # TODO: do something better than a ten-second delay to detect "boot:" prompt
        bootline = ("install netcfg/get_ipaddress=%s homeworld/asktoken=%s\n" % (self.node.ip, bootstrap_token)).encode()
        if self.boot_with_io("install", text=bootline, delay=10.0, chardelay=0.1).wait():
            command.fail("qemu virtual machine failed")

    def create_disk(self, size_gb=25):
        if os.path.exists(self.hd):
            os.remove(self.hd)
        if not os.path.isdir(os.path.dirname(self.hd)):
            os.makedirs(os.path.dirname(self.hd))
        subprocess.check_call(["qemu-img", "create", "-f", "qcow2", "--", self.hd, "%uG" % int(size_gb)])


class QEMUSession:
    # stdout_file should be any object with write() and close() methods.
    def __init__(self, vm: VirtualMachine, stdin_iter, stdout_file):
        self.vm = vm

        if stdin_iter is None:
            stdin_mechanism = subprocess.DEVNULL
        else:
            stdin_mechanism = subprocess.PIPE
        qemu_args = vm.generate_qemu_args()
        if vm.debug_qemu:
            print("QEMU: $", " ".join("'%s'" % arg for arg in qemu_args))
        self.process = subprocess.Popen(qemu_args, stdin=stdin_mechanism, stdout=subprocess.PIPE, bufsize=0)

        # make sure that all launched processes will die if we do
        atexit.register(self.terminate)

        if stdin_iter is not None:
            threading.Thread(target=self._input_loop, args=(stdin_iter,)).start()
        threading.Thread(target=self._output_loop, args=(stdout_file,)).start()

        vm.tc.add_session(self)

    def _input_loop(self, stdin_iter):
        for data in stdin_iter:
            self.process.stdin.write(data)
            self.process.stdin.flush()
        self.process.stdin.close()

    def _output_loop(self, stdout_file):
        while True:
            data = self.process.stdout.read(4096)
            if not data:
                break
            stdout_file.write(data)
            stdout_file.flush()
        stdout_file.close()

    def wait(self):
        rc = self.process.wait()
        self.vm.tc.remove_session(self)
        return rc

    def kill(self):
        if self.process.stdout:
            self.process.stdout.close()
        if self.process.stdin:
            self.process.stdin.close()
        self.process.kill()

    def terminate(self):
        if self.process.stdout:
            self.process.stdout.close()
        if self.process.stdin:
            self.process.stdin.close()
        self.process.terminate()


class SplitWriter:
    def __init__(self, *streams):
        self.streams = streams

    def write(self, data):
        for stream in self.streams:
            stream.write(data)

    def flush(self):
        for stream in self.streams:
            stream.flush()

    def close(self):
        for stream in self.streams:
            stream.close()


class LineCollector:
    def __init__(self, line_cb):
        self.line_cb = line_cb
        self.buffer = b""

    def write(self, bytestring):
        self.buffer += bytestring
        while b"\n" in self.buffer:
            segment, self.buffer = self.buffer.split(b"\n", 1)
            self.line_cb(segment + b"\n")

    def flush(self):
        pass

    def close(self):
        if self.buffer:
            self.line_cb(self.buffer)
            self.buffer = b""


def delayed_yielder(delay, chardelay, value: bytes):
    time.sleep(delay)
    first = True
    for c in value:
        if first:
            first = False
        else:
            time.sleep(chardelay)
        yield bytes([c])


class FingerprintExtractor:
    def __init__(self, callback):
        self.processing = True
        self.fingerprints = []
        self.callback = callback
        self.event = threading.Event()

    def process_line(self, line: bytes):
        if self.processing:
            if b"SHA256" in line and b" root@temporary-hostname (" in line:
                # TODO: don't just arbitrarily replace the string; parse and do a better conversion (for robustness)
                self.fingerprints.append(line.strip().decode().replace(" root@temporary-hostname ", " no comment "))
            elif self.fingerprints and not line.strip():
                self.callback(self.fingerprints)
                self.processing = False
                self.event.set()

    def wait(self):
        self.event.wait()


def qemu_check_nested_virt():
    if util.readfile("/sys/module/kvm_intel/parameters/nested").strip() != b"Y":
        command.fail("nested virtualization not enabled")


@command.wrapseq
def auto_install_supervisor(ops: command.Operations, tc: TerminationContext, supervisor: configuration.Node, install_iso: str, cdrom_install: bool=False, debug_qemu=False):
    'install supervisor node'
    vm = VirtualMachine(supervisor, tc, install_iso, cdrom_install=cdrom_install, debug_qemu=debug_qemu)
    ops.add_operation("install supervisor node (this may take several minutes)", vm.boot_install_supervisor, supervisor)


@command.wrapseq
def auto_launch_supervisor(ops: command.Operations, tc: TerminationContext, supervisor: configuration.Node, debug_qemu=False):
    'launch supervisor node'
    vm = VirtualMachine(supervisor, tc, debug_qemu=debug_qemu)
    ops.add_operation("start up supervisor node", lambda: vm.boot_launch(autoadd_fingerprint=True))


@command.wrapseq
def auto_install_nodes(ops: command.Operations, tc: TerminationContext, nodes: list, install_iso: str, cdrom_install: bool=False, debug_qemu=False):
    'install non-supervisor nodes'
    vms = [VirtualMachine(node, tc, install_iso, cdrom_install=cdrom_install, debug_qemu=debug_qemu) for node in nodes]

    def boot_install_and_admit_all():
        with concurrent.futures.ThreadPoolExecutor(len(vms)) as executor:
            futures = [executor.submit(vm.boot_install_and_admit) for vm in vms]
            for f in futures:
                f.result()

    ops.add_operation("install non-supervisor nodes (this may take several minutes)", boot_install_and_admit_all)


@command.wrapseq
def auto_launch_nodes(ops: command.Operations, tc: TerminationContext, nodes: list, debug_qemu=False):
    'launch non-supervisor nodes'
    for node in nodes:
        vm = VirtualMachine(node, tc, debug_qemu=debug_qemu)
        ops.add_operation("start up node {}".format(node), vm.boot_launch)


@command.wrapseq
def auto_install(ops: command.Operations, authorized_key=None, persistent: bool=False, cdrom_install: bool=False, debug_qemu: bool=False):
    "complete cluster installation and launch"
    if authorized_key is None:
        if "HOME" not in os.environ:
            command.fail("expected $HOME to be set for authorized_key autodetect")
        authorized_key = os.path.join(os.getenv("HOME"), ".ssh/id_rsa.pub")
    project, config = configuration.get_project(), configuration.get_config()
    iso_path = os.path.join(project, "cluster-%d.iso" % os.getpid())
    ops.add_operation("check nested virtualization", qemu_check_nested_virt)
    ops.add_operation("update known hosts", access.update_known_hosts)
    ops.add_operation("generate ISO", lambda: iso.gen_iso(iso_path, authorized_key, "serial"))
    with ops.context("networking", net_context()):
        with ops.context("termination", TerminationContext()) as tc:
            with ops.context("debug shell", DebugContext(persistent)):
                ops.add_subcommand(auto_install_supervisor, tc, config.keyserver, iso_path, cdrom_install=cdrom_install, debug_qemu=debug_qemu)
                ops.add_subcommand(auto_launch_supervisor, tc, config.keyserver, debug_qemu=debug_qemu)
                ops.add_subcommand(seq.sequence_supervisor)

                other_nodes = [n for n in config.nodes if n != config.keyserver]
                ops.add_subcommand(auto_install_nodes, tc, other_nodes, iso_path, cdrom_install=cdrom_install, debug_qemu=debug_qemu)
                ops.add_subcommand(auto_launch_nodes, tc, other_nodes, debug_qemu=debug_qemu)

                ops.add_subcommand(seq.sequence_cluster)


@command.wrapseq
def auto_launch(ops: command.Operations, debug_qemu: bool=False):
    "launch installed cluster"
    config = configuration.get_config()
    with ops.context("networking", net_context()):
        with ops.context("termination", TerminationContext()) as tc:
            with ops.context("debug shell", DebugContext(True)):
                ops.add_subcommand(auto_launch_supervisor, tc, config.keyserver, debug_qemu=debug_qemu)
                other_nodes = [n for n in config.nodes if n != config.keyserver]
                ops.add_subcommand(auto_launch_nodes, tc, other_nodes, debug_qemu=debug_qemu)


main_command = command.Mux("commands to run local testing VMs", {
    "net": command.Mux("commands to control the state of the local testing network", {
        "up": net_up,
        "down": net_down,
    }),
    "auto": command.SeqMux("commands to perform large-scale operations automatically", {
        "install": auto_install,
        "launch": auto_launch,
    }),
})

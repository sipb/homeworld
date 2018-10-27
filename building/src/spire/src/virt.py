import atexit
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
import setup
import util


def get_bridge(ip):
    return "spirebr%s" % hex(ip.to_integer())[2:].upper()


def get_node_tap(node):
    # maximum length: 15 characters
    return "spirtap%s" % hex(node.ip.to_integer())[2:].upper()


def determine_topology():
    config = configuration.get_config()
    gateway_ip = config.cidr_nodes.gateway()
    gateway = "%s/%d" % (gateway_ip, config.cidr_nodes.bits)
    taps = []
    hosts = {}
    for node in config.nodes:
        if node.ip not in config.cidr_nodes:
            command.fail("invalid topology: address %s is not in CIDR %s" % (node.ip, config.cidr_nodes))
        taps.append(get_node_tap(node))
        hosts["%s.%s" % (node.hostname, config.external_domain)] = node.ip
    return gateway, taps, get_bridge(gateway_ip), hosts


def sudo(*command):
    subprocess.check_call(["sudo"] + list(command))


def sudo_ok(*command):
    return subprocess.call(["sudo"] + list(command)) == 0


def sysctl_set(key, value):
    sudo("sysctl", "-w", "--", "%s=%s" % (key, value))


def bridge_up(bridge_name, address):
    sudo("brctl", "addbr", bridge_name)
    sudo("ip", "link", "set", bridge_name, "up")
    sudo("ip", "addr", "add", address, "dev", bridge_name)


def bridge_down(bridge_name, address):
    ok = sudo_ok("ip", "addr", "del", address, "dev", bridge_name)
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


def routing_up(bridge_name, upstream_link):
    sudo("iptables", "-I", "INPUT", "1", "-i", bridge_name, "-j", "ACCEPT")
    sudo("iptables", "-I", "FORWARD", "1", "-i", bridge_name, "-o", upstream_link, "-j", "ACCEPT")
    sudo("iptables", "-I", "FORWARD", "1", "-i", upstream_link, "-o", bridge_name, "-j", "ACCEPT")
    sudo("iptables", "-I", "FORWARD", "1", "-i", bridge_name, "-o", bridge_name, "-j", "ACCEPT")
    sudo("iptables", "-t", "nat", "-I", "POSTROUTING", "1", "-o", upstream_link, "-j", "MASQUERADE")


def routing_down(bridge_name, upstream_link):
    ok = sudo_ok("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", upstream_link, "-j", "MASQUERADE")
    ok &= sudo_ok("iptables", "-D", "FORWARD", "-i", bridge_name, "-o", bridge_name, "-j", "ACCEPT")
    ok &= sudo_ok("iptables", "-D", "FORWARD", "-i", upstream_link, "-o", bridge_name, "-j", "ACCEPT")
    ok &= sudo_ok("iptables", "-D", "FORWARD", "-i", bridge_name, "-o", upstream_link, "-j", "ACCEPT")
    ok &= sudo_ok("iptables", "-D", "INPUT", "-i", bridge_name, "-j", "ACCEPT")
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


def net_up_inner(gateway_ip, taps, bridge_name, hosts):
    upstream_link = get_upstream_link()

    sysctl_set("net.ipv4.ip_forward", 1)

    try:
        bridge_up(bridge_name, gateway_ip)
        for tap in taps:
            tap_up(bridge_name, tap)

        routing_up(bridge_name, upstream_link)

        hosts_up(hosts)
    except Exception as e:
        print("woops, tearing down...")
        if not net_down():
            print("could not tear down")
        raise e


def net_down_inner(gateway_ip, taps, bridge_name, hosts, fail=False):
    upstream_link = get_upstream_link()

    hosts_down(hosts)

    ok = routing_down(bridge_name, upstream_link)

    for tap in taps:
        ok &= tap_down(bridge_name, tap)

    ok &= bridge_down(bridge_name, gateway_ip)

    if not ok and fail:
        command.fail("tearing down network failed (maybe it was already torn down?)")
    return ok


def net_up():
    gateway_ip, taps, bridge_name, hosts = determine_topology()
    net_up_inner(gateway_ip, taps, bridge_name, hosts)


def net_down(fail=False):
    gateway_ip, taps, bridge_name, hosts = determine_topology()
    return net_down_inner(gateway_ip, taps, bridge_name, hosts, fail)


@contextlib.contextmanager
def net_context():
    gateway_ip, taps, bridge_name, hosts = determine_topology()
    net_up_inner(gateway_ip, taps, bridge_name, hosts)
    try:
        yield
    finally:
        net_down_inner(gateway_ip, taps, bridge_name, hosts, fail=True)


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
    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        if exc_type is not None:
            self.debug_shell()

    def debug_shell(self):
        print("launching debug shell...")
        subprocess.call(["bash"])
        print("debug shell quit")


class VirtualMachine:
    def __init__(self, node, tc: TerminationContext, cd=None, cpus=12, memory=2500):
        self.node = node
        self.cd = cd
        self.cpus = cpus
        self.memory = memory
        self.netif = get_node_tap(node)
        self.tc = tc

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
        args += ["-hda", self.hd]
        if self.cd is None:
            args += ["-boot", "c"]
        else:
            args += ["-cdrom", self.cd]
            args += ["-boot", "d"]
        args += ["-no-reboot"]
        args += ["-smp", "%d" % int(self.cpus), "-m", "%d" % int(self.memory)]
        if self.netif is None:
            args += ["-net", "none"]
        else:
            args += ["-net", "nic,macaddr=%s" % self.macaddress]
            args += ["-net", "tap,ifname=%s,script=no,downscript=no" % self.netif]
        return args

    def check_module_loaded(self):
        # TODO: don't blindly load kvm-intel; check type of system first
        sudo("modprobe", "kvm", "kvm-intel")

    def boot_with_io(self, phase, output_callback=None, text: bytes=None, delay=None):
        self.check_module_loaded()

        log_output = open("log.%s.%s" % (self.node.hostname, phase), "wb")
        try:
            if output_callback:
                output = SplitWriter(log_output, LineCollector(output_callback))
            else:
                output = log_output

            if text:
                stdin = delayed_yielder(delay, text)
            else:
                stdin = None

            return QEMUSession(self, stdin, output)
        except BaseException as e:
            log_output.close()
            raise e

    def boot_launch(self, autoadd_fingerprint=False):
        if not autoadd_fingerprint:
            return self.boot_with_io("launch")
        else:
            extractor = FingerprintExtractor(access.pull_supervisor_key)
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
        # TODO: do something better than a two-second delay to detect "boot:" prompt
        bootline = ("install netcfg/get_ipaddress=%s homeworld/asktoken=%s\n" % (self.node.ip, bootstrap_token)).encode()
        if self.boot_with_io("install", text=bootline, delay=2.0).wait():
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
        self.process = subprocess.Popen(vm.generate_qemu_args(),
                                        stdin=stdin_mechanism, stdout=subprocess.PIPE, bufsize=0)

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


def delayed_yielder(delay, value: bytes):
    time.sleep(delay)
    yield value


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


def auto_supervisor(ops: setup.Operations, tc: TerminationContext, supervisor: configuration.Node, install_iso: str):
    vm = VirtualMachine(supervisor, tc, install_iso)
    ops.add_operation("install supervisor node (this may take several minutes)", vm.boot_install_supervisor, supervisor)

    # TODO: annotations, so that this can be --dry-run'd
    vm = VirtualMachine(supervisor, tc)
    ops.add_operation("start up supervisor node", lambda: vm.boot_launch(autoadd_fingerprint=True))
    ops.add_subcommand(seq.sequence_supervisor)


def auto_node(ops: setup.Operations, tc: TerminationContext, node: configuration.Node, install_iso: str):
    vm = VirtualMachine(node, tc, install_iso)
    ops.add_operation("install node @HOST (this may take several minutes)", vm.boot_install_and_admit, node)
    vm = VirtualMachine(node, tc)
    ops.add_operation("start up node @HOST", lambda: vm.boot_launch(), node)


def auto_cluster(ops: setup.Operations, authorized_key=None):
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
            with ops.context("debug shell", DebugContext()):
                ops.add_subcommand(lambda ops: auto_supervisor(ops, tc, config.keyserver, iso_path))
                for node in config.nodes:
                    if node == config.keyserver: continue
                    ops.add_subcommand(lambda ops, n=node: auto_node(ops, tc, n, iso_path))

                ops.add_subcommand(seq.sequence_cluster)


main_command = seq.seq_mux_map("commands to run local testing VMs", {
    "net": command.mux_map("commands to control the state of the local testing network", {
        "up": command.wrap("bring up local testing network", net_up),
        "down": command.wrap("bring down local testing network", net_down),
    }),
    "auto": seq.seq_mux_map("commands to perform large-scale operations automatically", {
        "cluster": seq.wrapseq("complete cluster installation", auto_cluster),
    }),
})

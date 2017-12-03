import command
import tempfile
import access
import configuration
import util
import os


def launch_spec(spec_name):
    with tempfile.TemporaryDirectory() as d:
        specfile = os.path.join(d, "spec.yaml")
        util.writefile(specfile, configuration.get_single_kube_spec(spec_name).encode())
        access.call_kubectl(["create", "-f", specfile], return_result=False)


main_command = command.mux_map("commands to deploy systems onto the kubernetes cluster", {
    "flannel": command.wrap("deploy the specifications to run flannel", lambda: launch_spec("flannel.yaml")),
    "dns-addon": command.wrap("deploy the specifications to run the dns-addon", lambda: launch_spec("dns-addon.yaml")),
})

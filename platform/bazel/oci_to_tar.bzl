# Based on container/push.bzl from rules_docker
# Also based on pkg/pkg.bzl from bazel_tools

# Copyright 2015, 2017 The Bazel Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("@io_bazel_rules_docker//container:layer_tools.bzl", _get_layers = "get_from_target")

def _quote(filename, protect = "="):
    """Quote the filename, by escaping = by \= and \ by \\"""
    return filename.replace("\\", "\\\\").replace(protect, "\\" + protect)

def _impl(ctx):
    image = _get_layers(ctx, ctx.label.name, ctx.attr.image)
    blobsums = image.get("blobsum", [])
    blobs = image.get("zipped_layer", [])
    config = image["config"]
    manifest = image["manifest"]
    tarball = image.get("legacy")

    base = ctx.attr.folder
    mapping = {}

    if tarball:
        print("Pushing an image based on a tarball can be very " +
              "expensive.  If the image is the output of a " +
              "docker_build, consider dropping the '.tar' extension. " +
              "If the image is checked in, consider using " +
              "docker_import instead.")
        mapping[base + "/tarball"] = tarball
    if config:
        mapping[base + "/config"] = config
    if manifest:
        mapping[base + "/manifest"] = manifest
    for i, f in enumerate(blobsums):
        mapping[base + "/digest." + str(i)] = f
    for i, f in enumerate(blobs):
        mapping[base + "/layer." + str(i)] = f

    # Start building the arguments.
    args = [
        "--output=" + ctx.outputs.out.path,
        "--directory=/",
        "--mode=0644",
        "--owner=0.0",
        "--owner_name=.",
    ]

    file_inputs = []

    for f_dest_path, target in mapping.items():
        target_files = [target] # .files.to_list()
        if len(target_files) != 1:
            fail("Each input must describe exactly one file.", attr = "files")
        file_inputs += target_files
        args += ["--file=%s=%s" % (_quote(target_files[0].path), f_dest_path)]

    arg_file = ctx.actions.declare_file(ctx.label.name + ".args")
    ctx.actions.write(arg_file, "\n".join(args))

    ctx.actions.run(
        inputs = file_inputs + [arg_file],
        executable = ctx.executable.build_tar,
        arguments = ["--flagfile", arg_file.path],
        outputs = [ctx.outputs.out],
        mnemonic = "TarOCI",
        use_default_shell_env = True,
    )

oci_to_tar = rule(
    attrs = {
        "folder": attr.string(
            mandatory = True,
        ),
        "image": attr.label(
            allow_single_file = [".tar"],
            mandatory = True,
            doc = "The label of the image to push.",
        ),
        # Implicit dependencies.
        "build_tar": attr.label(
            default = Label("@bazel_tools//tools/build_defs/pkg:build_tar"),
            cfg = "host",
            executable = True,
            allow_files = True,
        ),
    },
    implementation = _impl,
    outputs = {
        "out": "%{name}.tar",
    },
)

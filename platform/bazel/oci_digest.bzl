# Modified from code under the following copyright:
# Copyright 2017 The Bazel Authors. All rights reserved.
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

# based on _assemble_image_digest from upstream
# the difference between this and the normal <name>.digest property that upstream provides is that this reports *OCI*
# digests, not *docker* digests. OCI digests are needed for the OCI distribution v2 API to work.

def _impl(ctx):
    output = ctx.outputs.digest
    image = _get_layers(ctx, ctx.label.name, ctx.attr.image)
    blobsums = image.get("blobsum", [])
    blobs = image.get("zipped_layer", [])

    arguments = [
        "--oci",
        "--config=%s" % image["config"].path,
        "--output-digest=%s" % output.path,
    ]
    arguments += ["--layer=%s" % f.path for f in blobs]
    arguments += ["--digest=%s" % f.path for f in blobsums]

    tools = [
        image["config"],
    ]
    tools += blobs
    tools += blobsums

    if image.get("legacy"):
        arguments += ["--tarball=%s" % image["legacy"].path]
        tools += [image["legacy"]]

    ctx.actions.run(
        outputs = [output],
        tools = tools,
        executable = ctx.executable._digester,
        arguments = arguments,
        mnemonic = "ImageDigest",
        progress_message = "Extracting image digest of %s" % ctx.attr.image,
    )

oci_digest = rule(
    attrs = {
        "image": attr.label(
            allow_single_file = [".tar"],
            mandatory = True,
        ),
        # implicit dependency!
        "_digester": attr.label(
            default = "@containerregistry//:digester",
            cfg = "host",
            executable = True,
        ),
    },
    implementation = _impl,
    outputs = {
        "digest": "%{name}.digest",
    },
)

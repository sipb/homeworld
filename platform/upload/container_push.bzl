load("//bazel:package.bzl", "homeworld_deb")
load("//bazel:oci_to_tar.bzl", "oci_to_tar")

def oci_pusher(name, packagebase, images, visibility = None):
    depends = [
        "python",
    ]
    packages = [name]

    for pkgname, label in images.items():
        tar = name + ".tar." + pkgname
        deb = name + ".deb." + pkgname
        subpackage = packagebase + "-" + pkgname
        folder = "/usr/lib/homeworld/ocis/" + pkgname

        oci_to_tar(
            name = tar,
            folder = folder,
            image = label,
        )
        homeworld_deb(
            name = deb,
            package = subpackage,
            deps = [tar],
            visibility = visibility,
        )
        depends += [subpackage]
        packages += [deb]

    homeworld_deb(
        name = name,
        package = packagebase,
        bin = {
            "//upload:src/push-ocis.sh": "/usr/lib/homeworld/push-ocis.sh",
            "@containerregistry//:pusher.par": "/usr/lib/homeworld/pusher.par",
        },
        depends = depends,
        visibility = visibility,
    )

    return packages

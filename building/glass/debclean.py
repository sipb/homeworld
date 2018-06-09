import os
import shutil


def clean_paths(rootfs, *paths):
    for path in paths:
        abspath = os.path.join(rootfs, path.lstrip("/"))
        if os.path.exists(abspath):
            if os.path.isdir(abspath):
                shutil.rmtree(abspath)
            else:
                os.unlink(abspath)


def clean_apt_files(rootfs):
    clean_paths(rootfs,
                "/var/cache/apt/", "/var/lib/apt/",
                "/var/log/bootstrap.log", "/var/log/alternatives.log", "/var/log/dpkg.log")


def clean_ld_aux(rootfs):
    clean_paths(rootfs, "/var/cache/ldconfig/aux-cache")


def clean_doc_files(rootfs):
    clean_paths(rootfs, "/usr/share/doc/", "/usr/share/man/")


def clean_locales(rootfs):
    localedir = os.path.join(rootfs, "usr/share/locale/")
    for subdir in os.listdir(localedir):
        if not subdir.startswith("en"):
            shutil.rmtree(os.path.join(localedir, subdir.lstrip("/")))


def clean_pycache(rootfs):
    clean_paths(rootfs,
                "/usr/lib/python3.5/unittest/__pycache__",
                "/usr/lib/python3.5/idlelib/__pycache__",
                "/usr/lib/python3.5/asyncio/__pycache__",
                "/usr/lib/python3.5/__pycache__")


def clean_resolv_conf(rootfs):
    clean_paths(rootfs, "/etc/resolv.conf")


DEBCLEAN_OPTIONS = {"apt_files": clean_apt_files, "ld_aux": clean_ld_aux, "doc_files": clean_doc_files,
                    "locales": clean_locales, "pycache": clean_pycache, "resolv_conf": clean_resolv_conf}

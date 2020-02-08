load("//bazel:substitute.bzl", "substitute")

def py_resources(name, data = None, visibility = None):
    substitute(
        name = "__init__.py",
        template = "//python:template_init.py",
    )
    native.filegroup(
        name = name,
        srcs = [":__init__.py"] + (data or []),
        visibility = visibility,
    )

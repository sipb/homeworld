
def _escape(s):
    return "'" + s.replace("'", "'\"'\"'") + "'"

def clean_paths(*paths):
    cmds = []
    for path in paths:
        cmds += ["rm -rf " + _escape(path.lstrip("/"))]
    return cmds

_OPTIONS = {
    "apt_files": clean_paths(
        "/var/cache/apt/",
        "/var/lib/apt/",
        "/var/log/bootstrap.log",
        "/var/log/alternatives.log",
        "/var/log/dpkg.log",
    ),
    "ld_aux": clean_paths(
        "/var/cache/ldconfig/aux-cache",
    ),
    "doc_files": clean_paths(
        "/usr/share/doc/",
        "/usr/share/man/",
    ),
    "locales": ["""
        for file in usr/share/locale/*
        do
            if [[ ! "$$(basename "$$file")" =~ ^en ]]
            then
                rm -rf $$file
            fi
        done
    """],
    "pycache": clean_paths(
       "/usr/lib/python3.5/unittest/__pycache__",
       "/usr/lib/python3.5/idlelib/__pycache__",
       "/usr/lib/python3.5/asyncio/__pycache__",
       "/usr/lib/python3.5/__pycache__",
    ),
    "resolv_conf": clean_paths(
        "/etc/resolv.conf",
    ),
}

def debclean(name, partial, clean_opts, visibility=None):
    """Clean out some of the large, host-dependent, or unnecessary files from a debian installation."""

    cmds = [
        "ORIG=\"$$PWD\"",
        "BTEMP=\"$$(mktemp -d)\"",
        "tar -xzf '$<' -C \"$${BTEMP}\"",
        "cd \"$${BTEMP}\"",
    ]
    for option in clean_opts:
        if option not in _OPTIONS:
            fail("invalid debclean option: " + option)
        cmds += _OPTIONS[option]
    cmds += [
        "cd \"$${ORIG}\"",
        "tar -czf '$@' --hard-dereference -C \"$${BTEMP}\" $$(ls -A \"$${BTEMP}\")",
        "rm -rf \"$${BTEMP}\"",
    ]
    native.genrule(
        name = name + "-rule",
        outs = [name],
        srcs = [partial],
        local = 1,
        cmd = "\n".join(cmds),
        visibility = visibility
    )

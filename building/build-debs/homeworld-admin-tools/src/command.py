import inspect


class CommandFailedException(Exception):
    pass


def fail(message: str) -> None:
    raise CommandFailedException(str(message))


def mux_map(desc: str, mapping: dict):
    def usage() -> None:
        print("commands:")
        for name, (desc, subinvoke) in sorted(mapping.items()):
            print("  %s: %s" % (name, desc))

    def invoke(params):
        if not params:
            usage()
            print("no command")
        elif params[0] not in mapping:
            usage()
            fail("unknown command: %s" % params[0])
        else:
            desc, subinvoke = mapping[params[0]]
            subinvoke(params[1:])

    if "usage" not in mapping:
        mapping = dict(mapping)
        mapping["usage"] = ("ask for this usage info", lambda _: usage())

    return desc, invoke


def get_argcount(func) -> (int, int):
    argcount = func.__code__.co_argcount
    optionals = len(func.__defaults__) if func.__defaults__ else 0
    lower_bound, upper_bound = argcount - optionals, argcount
    if func.__code__.co_flags & inspect.CO_VARARGS:
        upper_bound = None
    return lower_bound, upper_bound


def wrap(desc: str, func, paramtx=None):
    minarg, maxarg = get_argcount(func)

    def invoke(params):
        if paramtx:
            prev = len(params)
            params, on_end = paramtx(params)
            rel = len(params) - prev
        else:
            on_end = None
            rel = 0
        if maxarg is None:
            expect = "%d-" % (minarg - rel)
        elif maxarg == minarg:
            expect = "%d" % (minarg - rel)
        else:
            expect = "%d-%d" % (minarg - rel, maxarg - rel)
        if len(params) < minarg:
            fail("not enough parameters (expected %s)" % expect)
        if maxarg is not None and len(params) > maxarg:
            fail("too many parameters (expected %s)" % expect)
        func(*params)
        if on_end:
            on_end()

    return desc, invoke


def main_invoke(command, params):
    desc, invoke = command
    try:
        invoke(params)
        return 0
    except CommandFailedException as e:
        print('\x1b[1;31m' + 'command failed: ' + str(e) + '\x1b[0m')
        return 1

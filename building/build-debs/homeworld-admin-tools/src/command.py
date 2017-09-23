
class CommandFailedException(Exception):
    pass


def fail(message: str) -> None:
    raise CommandFailedException(str(message))


def mux_map(desc: str, mapping: dict):
    def usage(err: str = "no command") -> None:
        print("commands:")
        for name, (desc, subinvoke) in mapping.items():
            print("  %s: %s" % (name, desc))
        if err is not None:
            fail(err)

    def invoke(params):
        if not params:
            usage()
        elif params[0] not in mapping:
            usage("unknown command: %s" % params[0])
        else:
            desc, subinvoke = mapping[params[0]]
            subinvoke(params[1:])

    if "usage" not in mapping:
        mapping = dict(mapping)
        mapping["usage"] = ("ask for this usage info", lambda _: usage(None))

    return desc, invoke


def get_argcount(func) -> (int, int):
    argcount = func.__code__.co_argcount
    optionals = len(func.__defaults__) if func.__defaults__ else 0
    return argcount - optionals, argcount


def wrap(desc: str, func):
    minarg, maxarg = get_argcount(func)

    def invoke(params):
        expect = ("%d" % minarg if minarg == maxarg else "%d-%d" % (minarg, maxarg))
        if len(params) < minarg:
            fail("not enough parameters (expected %s)" % expect)
        if len(params) > maxarg:
            fail("too many parameters (expected %s)" % expect)
        return func(*params)

    return desc, invoke


def main_invoke(command, params):
    desc, invoke = command
    try:
        invoke(params)
        return 0
    except CommandFailedException as e:
        print(e)
        return 1

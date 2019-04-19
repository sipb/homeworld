import inspect
import argparse


ANSI_ESCAPE_CODE_RED = "\x1b[1;31m"
ANSI_ESCAPE_CODE_YELLOW = "\x1b[1;33m"
ANSI_ESCAPE_CODE_RESET = "\x1b[1;0m"


class CommandFailedException(Exception):
    def __init__(self, message, hint):
        super().__init__(message)
        self.hint = hint


def fail(message: str, hint: str = None) -> None:
    raise CommandFailedException(message, hint)


def provide_command_for_function(f, command):
    if hasattr(f, "dispatch_set_name"):
        f.dispatch_set_name(command)
    f.dispatch_name = command


def get_command_for_function(f):
    default = ["<unannotated subcommand: %s>" % f]
    if hasattr(f, "dispatch_get_name"):
        return f.dispatch_get_name(default)
    if hasattr(f, "dispatch_name"):
        return f.dispatch_name
    return default


def mux_map(desc: str, mapping: dict):
    def configure(command: list, parser: argparse.ArgumentParser):
        parser.set_defaults(argparse_parser=parser)
        subparsers = parser.add_subparsers()

        for component, (inner_desc, inner_configure) in mapping.items():
            inner_parser = subparsers.add_parser(component, description=inner_desc, help=inner_desc)
            inner_configure(command + [component], inner_parser)

    return desc, configure


def get_argcount(func) -> (int, int):
    argcount = func.__code__.co_argcount
    optionals = len(func.__defaults__) if func.__defaults__ else 0
    lower_bound, upper_bound = argcount - optionals, argcount
    if func.__code__.co_flags & inspect.CO_VARARGS:
        upper_bound = None
    return lower_bound, upper_bound


def wrap(desc: str, func, paramtx=None):
    minarg, maxarg = get_argcount(func)

    def invoke(args):
        params = args.argparse_params
        if paramtx:
            prev = len(params)
            params, on_end = paramtx(args)
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
        varnames = func.__code__.co_varnames
        opts = vars(args)
        opts = { k: opts[k] for k in varnames if k in opts }
        func(*params, **opts)
        if on_end:
            on_end()

    def configure(command: list, parser: argparse.ArgumentParser):
        parser.set_defaults(argparse_invoke=invoke, argparse_parser=parser)
        parser.add_argument('argparse_params', nargs=argparse.REMAINDER, help=argparse.SUPPRESS)
        provide_command_for_function(func, command)

    return desc, configure


def main_invoke(command):
    desc, configure_parser = command
    parser = argparse.ArgumentParser(description="Administrative toolkit for deploying and maintaining Hyades clusters")
    configure_parser(["spire"], parser)
    try:
        args = parser.parse_args()
        if "argparse_invoke" in args:
            args.argparse_invoke(args)
        else:
            args.argparse_parser.print_help()
        return 0
    except CommandFailedException as e:
        print(ANSI_ESCAPE_CODE_RED + 'command failed: ' + str(e) + ANSI_ESCAPE_CODE_RESET)
        if e.hint is not None:
            print(ANSI_ESCAPE_CODE_YELLOW + e.hint + ANSI_ESCAPE_CODE_RESET)
        return 1

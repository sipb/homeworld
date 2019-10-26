import argparse
import contextlib
import functools
import inspect
import time


ANSI_ESCAPE_CODE_RED = "\x1b[1;31m"
ANSI_ESCAPE_CODE_YELLOW = "\x1b[1;33m"
ANSI_ESCAPE_CODE_RESET = "\x1b[1;0m"


class CommandFailedException(Exception):
    def __init__(self, message, hint):
        super().__init__(message)
        self.hint = hint

    def __str__(self):
        return '{}command failed: {}{}{}'.format(
            ANSI_ESCAPE_CODE_RED,
            super().__str__(),
            '\n{}{}'.format(ANSI_ESCAPE_CODE_YELLOW, self.hint)
            if self.hint is not None else '',
            ANSI_ESCAPE_CODE_RESET)

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


def wrap(desc: str, func, param_tx=None):
    sig = inspect.signature(func)

    def invoke(args):
        opts = vars(args)
        if param_tx:
            opts, on_end = param_tx(opts)
        else:
            on_end = None

        kwargs = {}
        posargs = []

        for _, p in sig.parameters.items():
            if p.kind == inspect.Parameter.POSITIONAL_OR_KEYWORD:
                if p.default == inspect.Parameter.empty:
                    posargs += [opts[p.name]]
                else:
                    if p.name in opts:
                        kwargs[p.name] = opts[p.name]
            elif p.kind == inspect.Parameter.VAR_POSITIONAL:
                if p.name in opts:
                    posargs += opts[p.name]
            else:
                raise Exception("python argument type not recognized during invoke")

        func(*posargs, **kwargs)
        if on_end is not None:
            on_end()

    def configure(command: list, parser: argparse.ArgumentParser):
        parser.set_defaults(argparse_invoke=invoke, argparse_parser=parser)

        # convert function signature into argparse configuration
        for _, p in sig.parameters.items():
            # TODO: make ops an optional argument in all wrapped functions
            # instead of the first required argument
            if p.name == "ops":
                continue

            if p.kind == inspect.Parameter.POSITIONAL_OR_KEYWORD:
                if p.default == inspect.Parameter.empty:
                    parser.add_argument(p.name)
                elif isinstance(p.default, bool):
                    if p.default:
                        raise Exception("arguments defaulting to True not supported")
                    parser.add_argument("--%s" % (p.name), action="store_true")
                else:
                    parser.add_argument(p.name, nargs='?', default=p.default)
            elif p.kind == inspect.Parameter.VAR_POSITIONAL:
                parser.add_argument(p.name, nargs=argparse.REMAINDER)
            else:
                raise Exception("python argument type not recognized during configure")

        provide_command_for_function(func, command)

    return desc, configure


def main_invoke(command):
    desc, configure_parser = command
    parser = argparse.ArgumentParser(description="Administrative toolkit for deploying and maintaining Hyades clusters")
    configure_parser(["spire"], parser)
    args = parser.parse_args()
    if "argparse_invoke" in args:
        args.argparse_invoke(args)
    else:
        args.argparse_parser.print_help()
    return 0

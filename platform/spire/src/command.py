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


class Mux:
    def __init__(self, description, mapping):
        self.__doc__ = description
        self.mapping = mapping

    def configure(self, command: list, parser: argparse.ArgumentParser):
        parser.set_defaults(argparse_parser=parser)
        subparsers = parser.add_subparsers()

        for component, subcommand in self.mapping.items():
            doc = inspect.getdoc(subcommand)
            short_doc = doc.split('\n')[0] if doc else None
            subparser = subparsers.add_parser(
                component,
                description=doc,
                help=short_doc,
                formatter_class=argparse.RawDescriptionHelpFormatter)
            try:
                subcommand.configure(command + [component], subparser)
            except AttributeError as e:
                raise Exception("error configuring subcommand {!r}".format(subcommand)) from e


class Command:
    def __init__(self, func):
        self.func = func
        self.sig = inspect.signature(self.func)
        self._remove_ops_from_sig()
        self._command = None

    def _remove_ops_from_sig(self):
        parameters = list(self.sig.parameters.values())
        if parameters[0].name != 'ops':
            raise ValueError('first argument to command must be ops')
        parameters = parameters[1:]
        self.sig = self.sig.replace(parameters=parameters)

    # so that this can still be called as the original function
    def __call__(self, *args, **kwargs):
        return self.func(*args, **kwargs)

    def operate(self, op, *args, **kwargs):
        "Schedule this command to be run by Operations"
        self.func(op, *args, **kwargs)

    def process_args(self, argparse_args):
        "Process command-line arguments into function arguments"
        cli_args = vars(argparse_args)
        posargs = []
        kwargs = {}
        for name, param in self.sig.parameters.items():
            if param.kind == inspect.Parameter.POSITIONAL_OR_KEYWORD:
                if param.default == inspect.Parameter.empty:
                    posargs.append(cli_args[name])
                    continue
                kwargs[name] = cli_args[name]
                continue
            if param.kind == inspect.Parameter.VAR_POSITIONAL:
                posargs.extend(cli_args[name])
                continue
            raise Exception("python argument type not recognized")

        # fail early if arguments do not match function signature
        self.sig.bind(*posargs, **kwargs)

        return posargs, kwargs

    def invoke(self, aargs):
        ops = Operations()
        args, kwargs = self.process_args(aargs)
        self.operate(ops, *args, **kwargs)
        ops()

    def configure(self, command: list, parser: argparse.ArgumentParser):
        parser.set_defaults(argparse_invoke=self.invoke,
                            argparse_parser=parser)

        # convert function signature into argparse configuration
        for name, param in self.sig.parameters.items():
            try:
                if param.kind == inspect.Parameter.POSITIONAL_OR_KEYWORD:
                    if param.annotation == bool:
                        if param.default == inspect.Parameter.empty or param.default:
                            raise ValueError("boolean argument must specify default value of false")
                        parser.add_argument('--' + name, action='store_true')
                        continue
                    if param.default == inspect.Parameter.empty:
                        parser.add_argument(name)
                        continue
                    if not (isinstance(param.default, str) or param.default is None):
                        raise ValueError("default for string argument must be string or None")
                    parser.add_argument('--' + name, default=param.default)
                    continue
                if param.kind == inspect.Parameter.VAR_POSITIONAL:
                    parser.add_argument(name, nargs=argparse.REMAINDER)
                    continue
                raise ValueError("python argument kind {} not recognized".format(param.kind))
            except Exception as e:
                raise Exception("command {}: failed to configure argument {}".format(command, name)) from e

        self._command = command

    def command(self, *args, **kwargs):
        "Produce a string representation of this command with the specified arguments"
        if self._command is None:
            return None

        bound = self.sig.bind(*args, **kwargs)
        cl = self._command[:]
        for k, v in bound.arguments.items():
            param = self.sig.parameters[k]
            if param.kind == inspect.Parameter.POSITIONAL_OR_KEYWORD:
                if param.default == inspect.Parameter.empty:
                    cl.append(str(v))
                    continue
                if param.annotation == bool:
                    if v:
                        cl.append('--{}'.format(k))
                    continue
                cl.append('--{}={}'.format(k, v))
                continue
            if param.kind == inspect.Parameter.VAR_POSITIONAL:
                cl.extend(str(x) for x in v)
                continue
            raise Exception("python argument type not recognized")
        return ' '.join(cl)


def wrapop(f):
    return functools.update_wrapper(Command(f), f, updated=[])


class Simple(Command):
    def _remove_ops_from_sig(self):
        pass

    def operate(self, op, *args, **kwargs):
        op.add_operation(self.__doc__, lambda: self.func(*args, **kwargs),
                         self.command(*args, **kwargs))

    def invoke(self, aargs):
        args, kwargs = self.process_args(aargs)
        self.func(*args, **kwargs)


def wrap(f):
    return functools.update_wrapper(Simple(f), f, updated=[])


def main_invoke(command):
    parser = argparse.ArgumentParser(description="Administrative toolkit for deploying and maintaining Hyades clusters")
    command.configure(["spire"], parser)
    args = parser.parse_args()
    if "argparse_invoke" in args:
        args.argparse_invoke(args)
    else:
        args.argparse_parser.print_help()
    return 0

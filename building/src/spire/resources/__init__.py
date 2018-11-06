import pkgutil


def get_resource(name: str) -> bytes:
    try:
        b = pkgutil.get_data(__package__, name)
    except OSError:
        raise Exception("no such embedded resource: %s" % name) from None
    if b is None:
        raise Exception("package cannot be located or loaded: %s" % __package__)
    assert type(b) == bytes  # todo: remove this
    return b

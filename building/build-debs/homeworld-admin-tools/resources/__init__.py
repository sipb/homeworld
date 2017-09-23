import pkgutil

def get_resource(name: str) -> bytes:
    b = pkgutil.get_data(__package__, name)
    if b is None:
        raise Exception("no such embedded resource: %s" % name)
    assert type(b) == bytes  # todo: remove this
    return b
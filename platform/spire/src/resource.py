import pkgutil
import util
from resources import get_resource


def copy_to(name: str, fileout: str) -> None:
    util.writefile(fileout, get_resource(name))

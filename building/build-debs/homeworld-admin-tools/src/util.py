import os
import subprocess


def readfile(filename: str) -> bytes:
    with open(filename, "rb") as f:
        return f.read()


def writefile(filename: str, data: bytes) -> None:
    with open(filename, "wb") as fo:
        fo.write(data)


def copy(filefrom: str, fileto: str) -> None:
    with open(filefrom, "rb") as fi:
        with open(fileto, "wb") as fo:
            while True:
                block = fi.read(4096)
                if not block: break
                fo.write(block)


def pwgen(length: int) -> bytes:
    assert type(length) == int and length > 0
    return subprocess.check_output(["pwgen", str(length), "1"]).strip()


def mkpasswd(password: bytes, hash: str="sha-512") -> bytes:
    return subprocess.check_output(["mkpasswd", "--stdin", "--method=%s" % hash], input=password).strip()

import subprocess
import command
import os


KEY_ENV = "HOMEWORLD_DISASTER"  # the disaster recovery key
CIPHER_ALGO = "AES256"


def get_keypath() -> str:
    key = os.getenv(KEY_ENV)
    if key is None:
        command.fail("no key specified in env var $HOMEWORLD_DISASTER")
    if not os.path.isfile(key):
        command.fail("no key found in file specified by $HOMEWORLD_DISASTER")
    return key


def gpg_encrypt_file(source_file: str, dest_file: str) -> None:
    keypath = get_keypath()

    encrypt_command = ["gpg", "--passphrase-file", keypath, "--symmetric", "--cipher-algo", CIPHER_ALGO,
                       "--output", dest_file, source_file]
    subprocess.check_call(encrypt_command)


def gpg_decrypt_in_memory(contents: bytes) -> bytes:
    keypath = get_keypath()

    decrypt_command = ["gpg", "--passphrase-file", keypath, "--decrypt"]
    return subprocess.check_output(decrypt_command, input=contents)


def gpg_decrypt_to_memory(source_file: str) -> bytes:
    keypath = get_keypath()

    decrypt_command = ["gpg", "--passphrase-file", keypath, "--decrypt", source_file]
    return subprocess.check_output(decrypt_command)


def gpg_decrypt_file(source_file: str, dest_file: str) -> None:
    keypath = get_keypath()

    encrypt_command = ["gpg", "--passphrase-file", keypath, "--decrypt", "--output", dest_file, source_file]
    subprocess.check_call(encrypt_command)

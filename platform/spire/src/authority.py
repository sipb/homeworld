import os

import command
import configuration
import util
import subprocess
import tempfile
import tarfile
import keycrypt

ENCRYPTED_EXTENSION = ".encrypted"


def get_targz_path(check_exists=True):
    authorities = os.path.join(configuration.get_project(), "authorities.tgz")
    if check_exists and not os.path.exists(authorities):
        command.fail("authorities.tgz does not exist (run spire authority gen?)")
    return authorities


def name_for_encrypted_file(filename):
    return filename + ENCRYPTED_EXTENSION


def name_for_decrypted_file(name_of_encrypted_file):
    if name_of_encrypted_file.endswith(ENCRYPTED_EXTENSION):
        return name_of_encrypted_file[:-len(ENCRYPTED_EXTENSION)]
    raise ValueError("Filename " + name_of_encrypted_file + " does not have expected suffix '" + ENCRYPTED_EXTENSION + "'.")


def validate_pem_file(full_file_name: str) -> bool:
    with open(full_file_name) as pem_file:
        has_certificate = False
        for line in pem_file:
            # PEM data that is a certificate should have the following header
            if line.startswith("-----BEGIN CERTIFICATE-----"):
                if has_certificate:
                    command.fail("pem file \"" + full_file_name + "\" should only contain 1 certificate, but it contains additional certificates!")
                has_certificate = True
            # The file should not have other types of PEM data
            elif line.startswith("-----BEGIN "):
                command.fail("pem file \"" + full_file_name + "\" contains non-certificate PEM data!")
        # The file should have had a certificate
        if not has_certificate:
            command.fail("pem file \"" + full_file_name + "\" does not contain any PEM certificates!")

        return True


def generate() -> None:
    authorities = get_targz_path(check_exists=False)
    if os.path.exists(authorities):
        command.fail("authorities.tgz already exists")
    # tempfile.TemporaryDirectory() creates the directory with 0o600, which protects the private keys
    with tempfile.TemporaryDirectory() as d:
        certdir = os.path.join(d, "certdir")
        keyserver_yaml = os.path.join(d, "keyserver.yaml")
        util.writefile(keyserver_yaml, configuration.get_keyserver_yaml().encode())
        os.mkdir(certdir)
        print("generating authorities...")
        try:
            # TODO: avoid having these touch disk
            subprocess.check_call(["keygen", keyserver_yaml, certdir])
        except FileNotFoundError as e:
            if e.filename == "keygen":
                command.fail("could not find keygen binary. is the homeworld-keyserver dependency installed?")
            else:
                raise e
        print("encrypting authorities...")
        cryptdir = os.path.join(d, "cryptdir")
        os.mkdir(cryptdir)
        for filename in os.listdir(certdir):
            if filename.endswith(".pub") or (filename.endswith(".pem") and validate_pem_file(os.path.join(certdir, filename))):
                # public keys; copy over without encryption
                util.copy(os.path.join(certdir, filename), os.path.join(cryptdir, filename))
            else:
                # private keys; encrypt when copying, and rename encrypted version for clarity.
                keycrypt.gpg_encrypt_file(os.path.join(certdir, filename), os.path.join(cryptdir, name_for_encrypted_file(filename)))
        subprocess.check_call(["shred", "--"] + os.listdir(certdir), cwd=certdir)
        print("packing authorities...")
        subprocess.check_call(["tar", "-C", cryptdir, "-czf", authorities, "."])
        subprocess.check_call(["shred", "--"] + os.listdir(cryptdir), cwd=cryptdir)


# this can be used for getting private keys, but it won't decrypt them for you
def get_pubkey_by_filename(keyname) -> bytes:
    authorities = get_targz_path()
    with tarfile.open(authorities, mode="r:gz") as tar:
        with tar.extractfile(keyname) as f:
            out = f.read()
            assert type(out) == bytes
            return out


def get_decrypted_by_filename(name) -> bytes:
    return keycrypt.gpg_decrypt_in_memory(get_pubkey_by_filename(name_for_encrypted_file(name)))


def iterate_keys():  # yields (name, contents) pairs
    authorities = get_targz_path()
    with tarfile.open(authorities, mode="r:gz") as tar:
        for member in tar.getmembers():
            if member.isreg():
                with tar.extractfile(member) as f:
                    contents = f.read()
                assert type(contents) == bytes
                if member.name.startswith("./"):
                    yield member.name[2:], contents
                else:
                    yield member.name, contents


def iterate_keys_decrypted():  # yields (name, contents) pairs
    for name, contents in iterate_keys():
        if name.endswith(".pub") or name.endswith(".pem"):
            yield name, contents
        else:
            yield name_for_decrypted_file(name), keycrypt.gpg_decrypt_in_memory(contents)


main_command = command.mux_map("commands about cluster authorities", {
    "gen": command.wrap("generate and encrypt authority keys and certs", generate),
})

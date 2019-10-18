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
        os.mkdir(certdir)
        print("generating authorities...")
        try:
            # TODO: avoid having these touch disk
            subprocess.check_call(["keygen", certdir])
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


def get_upstream_cert_paths():
    # we don't encrypt user-grant-upstream.key, because it's not intended to be saved
    return os.path.join(configuration.get_project(), "user-grant-upstream.key"), \
           os.path.join(configuration.get_project(), "user-grant-upstream.pem")


def get_local_grant_user_paths():
    # we don't encrypt local-grant-user.key, because it's not intended to be saved
    return os.path.join(configuration.get_project(), "local-grant-user.key"), \
           os.path.join(configuration.get_project(), "local-grant-user.pem")


def gen_local_upstream_user():
    config = configuration.get_config()
    if config.user_grant_email_domain == "":
        command.fail("user-grant-email-domain not populated when trying to generate local fake upstream certificate")
    ca_key, ca_cert = get_upstream_cert_paths()
    user_key, user_cert = get_local_grant_user_paths()
    if os.path.exists(ca_key) or os.path.exists(ca_cert):
        command.fail("upstream certificate authority already exists; not generating")
    if os.path.exists(user_key) or os.path.exists(user_cert):
        command.fail("locally-generated user already exists; not generating")
    print("warning: user-grant-upstream.key and local-grant-user.key will not be encrypted; you should not use these "
          "as long-term keys or commit them into your project repository. this feature is only intended for temporary "
          "cluster testing.")
    subprocess.check_call(["keygenupstream", ca_key, ca_cert,
                           "mortal@%s" % config.user_grant_email_domain, user_key, user_cert, "24h"])


main_command = command.mux_map("commands about cluster authorities", {
    "gen": command.wrap("generate and encrypt authority keys and certs", generate),
    "genupstream": command.wrap("generate a fake local user CA and sample user for testing", gen_local_upstream_user),
})

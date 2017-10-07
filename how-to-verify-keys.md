Getting cela's Public Key
========================

1. Get an _untrusted_ copy of cela's key signature from the Repository Security section in homeworld's `README.md`.
2. Get an _untrusted_ copy of cela's key from the MIT keyserver: `gpg --keyserver pgp.mit.edu --recv-keys <last 8 digits of untrusted cela's key signature>`
3. Compute the fingerprint of the received key: `gpg --fingerprint cela`
4. Ask cela for her fingerprint and check against the output of the previous command.
5. Set the trust on cela's key with `gpg --edit-key cela`, `trust`, `4`, `quit` (`5` is reserved for your own key).

Getting a Trusted Copy of `trust.txt`
=====================================

1. Clone the homeworld repository.
2. Get the signatures of the commits for `trust.txt`: `git log --show-signature trust.txt`
3. Check that the latest commit in the log is "good".

Verifying a Dependency's Signature
==================================

1. Download an _untrusted_ copy of the dependency's signing key (e.g. `dependency_pubkey.gpg`).
2. Import the dependency's signing key: `gpg --import <dependency_pubkey.gpg>`
3. Download an _untrusted_ copy of the dependency's source code (e.g. `dependency.tar.gz`) and the archive's signature (e.g. `dependency.tar.gz.asc`)
4. Verify the dependency's source code: `gpg --verify <dependency.tar.gz.asc> <dependency.tar.gz>`
5. Check that the fingerprint matches the one in `trust.txt`.
6. Congratulations! You have a verified copy of a dependency's source code!

Notes for Specific Dependencies
===============================

| Dependency | Verification Notes |
| ---------- | ------------------ |
| etcd | CoreOS application signing key: https://coreos.com/security/app-signing-key/ |
| flannel | Does not seem to have signed releases, waiting for confirmation from developers. |
| Go | Does not seem to have signed releases, waiting for confirmation from developers. |

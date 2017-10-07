Getting cela's Public Key
========================

1. Get an _untrusted_ copy of cela's key from the MIT keyserver: `gpg --keyserver pgp.mit.edu --recv-keys 0f825b91` (last 8 digits of cela's key fingerprint)
2. Compute the fingerprint of the received key: `gpg --fingerprint cela`
3. Ask cela for her fingerprint and check against the output of the previous command.

Getting a Trusted Copy of `trust.txt`
=====================================

1. Clone the homeworld repository.
2. Get the signatures of the commits for `trust.txt`: `git log --show-signature trust.txt`
3. Check that the latest commit in the log has a valid signature by someone trustworthy (in this case cela).

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

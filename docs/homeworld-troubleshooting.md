# Troubleshooting

This document is a list of tips and tricks useful for debugging different parts of Homeworld.

## Kubernetes Component Verbosity

When trying to figure out why a component of Kubernetes (kubelet, apiserver, et cetera) is misbehaving, it can be
helpful to turn up the verbosity on these components.

You can easily do this by creating the file `/etc/homeworld/config/verbosity` on the node you want to debug, and
putting the verbosity level (usually 0, 1, 2, or 3) into the file:

    $ echo 3 >/etc/homeworld/config/verbosity

This is processed by `kubernetes/launch.go`, and adds a `--v=N` parameter to the component. As such, you'll need to
restart the relevant service to see this take effect.


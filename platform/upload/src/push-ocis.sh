#!/bin/bash
set -e -u

# This file runs on the supervisor, and populates the registry with data already made available.

REGISTRY="localhost:580"  # the hardcoded port in docker-registry

for container in /usr/lib/homeworld/ocis/*
do
    ARGS=("--oci")
    if [ -e "${container}/tarball" ]
    then
        ARGS+=("--tarball=${container}/tarball")
    fi
    if [ -e "${container}/config" ]
    then
        ARGS+=("--config=${container}/config")
    fi
    if [ -e "${container}/manifest" ]
    then
        ARGS+=("--manifest=${container}/manifest")
    fi
    for digest in $(ls -v "${container}/digest."*)
    do
        ARGS+=("--digest=${digest}")
    done
    for layer in $(ls -v "${container}/layer."*)
    do
        ARGS+=("--layer=${layer}")
    done
    # this is a workaround. we should be able to reference images by digest without needing to make the digest into the
    # tag name, and it's more reliable like that, anyway.
    # but rkt + kubernetes are broken when used together with images referenced by digest. so we reuse the digest as a
    # tag, and for the time being use tag lookup when we shouldn't.
    TAG="$(cat "${container}/oci-digest")"
    ARGS+=("--name=${REGISTRY}/$(basename "${container}"):${TAG}")

    echo "pushing container ${container}"
    /usr/lib/homeworld/pusher.par "${ARGS[@]}"
done

echo "done pushing to registry"

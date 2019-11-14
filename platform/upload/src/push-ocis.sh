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
    while IFS='' read -r -d '' digest
    do
        ARGS+=("--digest=${digest}")
    done < <(find "${container}" -maxdepth 1 -type f -name 'digest.*' -print0 | sort -zV)
    while IFS='' read -r -d '' layer
    do
        ARGS+=("--layer=${layer}")
    done < <(find "${container}" -maxdepth 1 -type f -name 'layer.*' -print0 | sort -zV)
    # we don't reference anything by tag; simply push everything to a pointless tag that we don't care about.
    TAG="last"
    ARGS+=("--name=${REGISTRY}/$(basename "${container}"):${TAG}")

    echo "pushing container ${container}"
    /usr/lib/homeworld/pusher.par "${ARGS[@]}"
done

echo "done pushing to registry"

#!/bin/bash
set -e -u
if test -z "$(git status --untracked=no --porcelain)"; then
    dirty=""
else
    dirty="-dirty"
fi

echo "STABLE_GIT_COMMIT $(git rev-parse HEAD)$dirty"

#!/bin/bash
set -euo pipefail

if [ 0 = "$(git rev-list --min-parents=2 --count "$(git merge-base origin/master HEAD)"..HEAD)" ]
then
    echo 'git history is linear'
else
    echo 'error: nonlinear branch git history'
    echo 'merge commits:'
    git rev-list --min-parents=2 "$(git merge-base origin/master HEAD)"..HEAD
    exit 1
fi

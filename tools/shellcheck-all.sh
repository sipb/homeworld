#!/bin/bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

failed=""
while IFS="" read -r -d '' file
do
    if grep -Eq '^#!(.*/|.*env +)(sh|bash)' "$file" || [[ "$file" =~ \.(ba)?sh$ ]]
    then
        # SC1091: Some of our scripts include configuration files which are not known statically.
        # TODO(#431): re-enable SC1091
        # SC2015: Overzealous rule which forbids reasonable code
        # SC2016: Rule matches on user-friendly hints to set variables
        # as well as on nested code snippets passed to shells.
        if ! shellcheck -e SC1091,SC2015,SC2016 "$file"
        then
            failed=failed
        fi
    fi
done < <(git ls-files -z)
test -z "$failed"

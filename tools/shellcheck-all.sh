#!/bin/bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

failed=""
while IFS="" read -r file
do
    if grep -Eq '^#!(.*/|.*env +)(sh|bash)' "$file" || [[ "$file" =~ \.(ba)?sh$ ]]
    then
        if ! shellcheck "$file"
        then
            failed=failed
        fi
    fi
done < <(git ls-files)
test -z "$failed"

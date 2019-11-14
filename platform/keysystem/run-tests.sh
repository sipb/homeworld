#!/bin/bash
set -e -u

cd "$(dirname "$0")"

find -wholename '*/testdir/inaccessible' -type d -exec rmdir {} \; 2>/dev/null || true
find -wholename '*/testdir/nonexistent' -type d -exec rmdir {} \; 2>/dev/null || true
find -wholename '*/testdir/brokendir' -type d -exec rmdir {} \; 2>/dev/null || true

pushd src
TESTDIRS=()
while IFS='' read -r -d '' dir
do
    if compgen -G "$dir"/'*.go'
    then
        TESTDIRS+=("$dir")
    fi
done < <(find ./* -type d -print0)
popd

GOPATH="$GOPATH:$(pwd)"
export GOPATH

go test -cover "${TESTDIRS[@]}"

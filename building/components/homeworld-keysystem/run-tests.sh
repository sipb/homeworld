#!/bin/bash
set -e -u

cd "$(dirname "$0")"

find -wholename '*/testdir/inaccessible' -type d -exec rmdir {} \; 2>/dev/null || true
find -wholename '*/testdir/nonexistent' -type d -exec rmdir {} \; 2>/dev/null || true
find -wholename '*/testdir/brokendir' -type d -exec rmdir {} \; 2>/dev/null || true

TESTDIRS="$(cd src && for x in $(find * -type d); do if [ "$(echo $x/*.go)" != $x'/*.go' ]; then echo $x; fi; done)"

export GOPATH="$GOPATH:$(pwd)"

go test -cover $TESTDIRS

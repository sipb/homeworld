#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

VERSION="0.1.0"

UPDATE_TIMESTAMP="2018-01-11T22:47:00-0500"

common_setup

gcc -static pullcheck.c -o "${B}/pullcheck"

start_acbuild
$ACBUILD copy "${B}/pullcheck" /usr/bin/pullcheck
$ACBUILD set-exec -- /usr/bin/pullcheck
finish_acbuild

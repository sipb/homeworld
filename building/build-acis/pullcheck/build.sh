#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

VERSION="0.1.0"

DEBVER="stretch.20180111T215606Z"
BUILDVER="stretch.20180111T215606Z"
UPDATE_TIMESTAMP="2018-01-11T22:47:00-0500"

common_setup
init_builder

cp pullcheck.c "${BUILDDIR}/pullcheck.c"
run_builder "gcc -static pullcheck.c -o pullcheck"

start_acbuild
$ACBUILD copy "${BUILDDIR}/pullcheck" /usr/bin/pullcheck
$ACBUILD set-exec -- /usr/bin/pullcheck
finish_acbuild

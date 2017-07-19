#!/bin/bash
set -e -u

if [[ "$EUID" != "0" ]]
then
	echo "Sudoing..."
	exec sudo "$0" "$@"
	exit 1
fi

if [ "$(uname -m)" != "x86_64" ]
then
	echo "Expecting to be on an amd64 system!" 1>&2
	exit 1
fi

ACBUILD_TGZ="../../build-helpers/acbuild-bin-0.4.0.tgz"

if [ ! -e "$ACBUILD_TGZ" ]
then
	echo "Could not find acbuild tar!" 1>&2
	exit 1
fi

ACI_BRIEF="debian"
ACI_NAME="hyades.mit.edu/homeworld/${ACI_BRIEF}"
RELEASE=stretch
VERSION="${RELEASE}.$(date '+%Y.%m.%d.%H')"
OUTPUT_FILE="homeworld-${ACI_BRIEF}-${VERSION}-linux-amd64.aci"
OUTPUT_DIR="../containers"

function acbuildend() {
	export EXIT="$?"
	$ACBUILD end && rm -rf rootfs && rm -rf acbuild && exit "$EXIT"
}

EXTRA="wget,curl,ca-certificates,git,realpath,file,less,gnupg,python,python3,iptables,iputils-ping,iputils-arping,iproute2,bzip2,gzip,net-tools,netcat-traditional,dnsutils"

rm -rf rootfs
mkdir rootfs
debootstrap --variant=minbase --components=main --include="$EXTRA" "${RELEASE}" rootfs http://mirrors.mit.edu/debian/
rm -rf rootfs/var/cache/apt/
rm -rf rootfs/var/lib/apt/

tar -xf "${ACBUILD_TGZ}" acbuild/acbuild
ACBUILD=acbuild/acbuild

$ACBUILD begin ./rootfs
trap acbuildend EXIT
$ACBUILD set-name "${ACI_NAME}"
$ACBUILD label add version "${VERSION}"
$ACBUILD set-working-dir "/"
$ACBUILD set-exec /bin/bash
$ACBUILD write --overwrite "${OUTPUT_FILE}"
mkdir -p "${OUTPUT_DIR}"
mv "${OUTPUT_FILE}" -t "${OUTPUT_DIR}"

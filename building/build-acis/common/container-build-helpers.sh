ROOT="$(pwd)"
UPSTREAM="${ROOT}/../../upstream"
HELPERS="${ROOT}/../../build-helpers"
OUTPUT_DIR="${ROOT}/../containers"

ACBUILD_VER=0.4.0
ACBUILD_TGZ="${HELPERS}/acbuild-bin-${ACBUILD_VER}.tgz"
ACBUILD=acbuild
ACBUILDDIR="${ROOT}/acbuild-tmp"

GO_VER=1.8.3
GO_TGZ="${HELPERS}/go-bin-${GO_VER}.tgz"

ACI_BRIEF="$(basename ${ROOT})"
ACI_NAME="hyades.mit.edu/homeworld/${ACI_BRIEF}"

function common_setup() {
	ensure_sudo
	ensure_amd64
	importacbuild
}

function ensure_sudo() {
	if [[ "$EUID" != "0" ]]
	then
		echo "Sudoing..."
		exec sudo "$0" "$@"
		exit 1
	fi
}

function ensure_amd64() {
	if [ "$(uname -m)" != "x86_64" ]
	then
		echo "Expecting to be on an amd64 system!" 1>&2
		exit 1
	fi
}

function importacbuild() {
	if [ ! -e "${ACBUILD_TGZ}" ]
	then
		echo "Could not find acbuild tar!" 1>&2
		exit 1
	fi
	rm -rf "${ACBUILDDIR}"
	mkdir "${ACBUILDDIR}"
	tar -C "${ACBUILDDIR}" -x --strip-components 1 -f "${ACBUILD_TGZ}" acbuild/acbuild
	ACBUILD="${ACBUILDDIR}/acbuild"
	if [ ! -x "${ACBUILD}" ]
	then
		echo "Failed to extract a working acbuild executable." 1>&2
		exit 1
	fi
}

function on_acbuildend() {
	export EXIT="$?"
	$ACBUILD end && exit "$EXIT"
}

function start_acbuild() {
	if [ "${VERSION}" = "" ]
	then
		echo "Version must be defined!" 1>&2
		exit 1
	fi
	mkdir -p "${OUTPUT_DIR}"
	$ACBUILD begin "$@"
	trap on_acbuildend EXIT
	$ACBUILD set-name "${ACI_NAME}"
	$ACBUILD label add version "${VERSION}"
}

function start_acbuild_from() {
	FROM="${OUTPUT_DIR}/homeworld-$1-$2-linux-amd64.aci"
	if [ ! -e "${FROM}" ]
	then
		echo "Could not find upstream container $1 at version $2" 1>&2
		exit 1
	fi
	start_acbuild "${FROM}"
}

function finish_acbuild() {
	$ACBUILD write --overwrite "${OUTPUT_DIR}/homeworld-${ACI_BRIEF}-${VERSION}-linux-amd64.aci"
	trap - EXIT
	on_acbuildend
}

# rkt builder

function init_builder() {
	BUILDENV="${OUTPUT_DIR}/homeworld-debian-build-${BUILDVER}-linux-amd64.aci"
	BUILDDIR="${ROOT}/build"
	rm -rf "${BUILDDIR}"
	mkdir "${BUILDDIR}"
	BUILDSCRIPT_GEN=(gen_base)
	BUILDPATH="/build"
}

function path_to_buildpath() {
	echo "/build/$(realpath "$1" "--relative-to=${BUILDDIR}")"
}

function gen_base() {
	echo "#!/bin/bash"
	echo "set -e -u"
	echo "cd '${BUILDPATH}'"
	echo "echo Beginning build within build container..."
}

function build_with_go() {
	if [ "${BUILDDIR}" = "" ] || [ ! -e "${BUILDDIR}" ]
	then
		echo "Invalid builddir setup." 1>&2
		exit 1
	fi
	if [ ! -e "${GO_TGZ}" ]
	then
		echo "Cannot find go binaries." 1>&2
		exit 1
	fi

	tar -C "${BUILDDIR}" -xf "${GO_TGZ}" go/

	BUILDSCRIPT_GEN+=(gen_go_setup)
}

function gen_go_setup() {
	echo 'export GOROOT="/build/go"'
	echo 'export PATH="$PATH:$GOROOT/bin"'
	echo "export GOPATH='$(path_to_buildpath "${GODIR}")'"
	echo 'if [ "$(go version 2>/dev/null)" != "go version go'"${GO_VER}"' linux/amd64" ]'
	echo 'then'
	echo "    echo 'go version mismatch! expected ${GO_VER}' 1>&2"
	echo "    go version 1>&2"
	echo "    exit 1"
	echo "fi"
}

function build_at_path() {
	BUILDPATH="$(path_to_buildpath "${1}")"
}

function run_builder() {
	(for generator in "${BUILDSCRIPT_GEN[@]}"
	do
		$generator
	done
	for line in "$@"
	do
		echo "$line"
	done) >${BUILDDIR}/inner-build.sh

	chmod +x ${BUILDDIR}/inner-build.sh

	# stage1 should not be kvm
	RKT_OPTS=(--stage1-path=/usr/lib/rkt/stage1-images/stage1-coreos.aci)

	# use the build environment container
	RKT_OPTS+=(--insecure-options=image "${BUILDENV}")

	# bind the build directory
	RKT_OPTS+=(--volume "build,kind=host,source=${BUILDDIR},readOnly=false")
	RKT_OPTS+=(--mount volume=build,target=/build)

	# run the script
	RKT_OPTS+=(--exec=/build/inner-build.sh)

	echo "Launching builder..."
	rkt run "${RKT_OPTS[@]}"
	echo "Build complete!"
}

if [ ! -e /h/ ]
then
	echo "expected to run within homeworld build chroot" 1>&2
	exit 1
fi

ROOT="$(pwd)"
UPSTREAM="${ROOT}/../../upstream"
HELPERS="${ROOT}/../../build-helpers"
OUTPUT_DIR="${ROOT}/../containers"

ACBUILD_VER=0.4.0
ACBUILD_TGZ="${HELPERS}/acbuild-bin-${ACBUILD_VER}.tgz"
ACBUILD=acbuild
ACBUILDDIR="${ROOT}/acbuild-tmp"

GO_VER=1.9.3
GO_TGZ="${HELPERS}/go-bin-${GO_VER}.tgz"

ACI_BRIEF="$(basename "${ROOT}")"
ACI_NAME="homeworld.mit.edu/${ACI_BRIEF}"

function common_setup() {
	ensure_sudo
	ensure_amd64
	add_exit_condition
	allocate_tempdir
	importacbuild
}

function common_onexit() {
	export EXIT="$?"
	if [ "${ISACBUILDING:-}" != "" ]
	then
		$ACBUILD end
	fi
	if [ "${TMPBUILDDIR:-}" != "" ] && [ -d "${TMPBUILDDIR}" ]
	then
		if [ "$EXIT" == 0 ]
		then
			rm -rf "${TMPBUILDDIR}"
		else
			echo "Not deleting ${TMPBUILDDIR} due to failure."
		fi
	fi
	exit $EXIT
}

function add_exit_condition() {
	trap common_onexit EXIT
}

function allocate_tempdir() {
	TMPBUILDDIR="$(mktemp -d)"
	if [ ! -d "${TMPBUILDDIR}" ]
	then
		echo "Could not create temporary directory." 1>&2
		exit 1
	fi
	B="${TMPBUILDDIR}"
}

function ensure_sudo() {
	if [[ "$EUID" != "0" ]]
	then
		echo "Sudoing..."
		exec sudo TMPDIR="${TMPDIR:-}" "$0" "$@"
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

function extract_upstream_as() {
	mkdir -p "${B}/extract"
	tar -C "${B}/extract" -xf "${UPSTREAM}/$1" "$2"
	mkdir -p "$(dirname "$3")"
	mv "${B}/extract/$2" -T "$3"
}

function start_acbuild() {
	if [ "${VERSION:-}" = "" ]
	then
		echo "Version must be defined!" 1>&2
		exit 1
	fi
	if [ "${UPDATE_TIMESTAMP:-}" = "" ]
	then
		echo "Update timestamp must be defined!" 1>&2
		exit 1
	fi
	mkdir -p "${OUTPUT_DIR}"
	$ACBUILD begin "$@"
	ISACBUILDING="yes"
	$ACBUILD set-name "${ACI_NAME}"
	$ACBUILD label add version "${VERSION}"
}

function start_acbuild_from() {
	FROM="${OUTPUT_DIR}/$1-$2-linux-amd64.aci"
	if [ ! -e "${FROM}" ]
	then
		echo "Could not find upstream container $1 at version $2" 1>&2
		exit 1
	fi
	start_acbuild "${FROM}"
}

function add_packages_to_acbuild() {
	$ACBUILD run -- apt-get update
	$ACBUILD run -- apt-get install -y "$@"
	$ACBUILD run -- rm -rf /var/log/dpkg.log /var/cache/apt /var/lib/apt /var/log/alternatives.log
}

function finish_acbuild() {
	ACI_OUTPUT="${OUTPUT_DIR}/${ACI_BRIEF}-${VERSION}-linux-amd64.aci"
	ACI_IMMD="${B}/homeworld-intermediate.aci"
	rm -f "${ACI_IMMD}"
	$ACBUILD write --overwrite "${ACI_IMMD}"
	ISACBUILDING=""
	$ACBUILD end
	ACI_REBUILD_TMP="${B}/acrebuild-tmp"
	rm -rf "${ACI_REBUILD_TMP}"
	mkdir "${ACI_REBUILD_TMP}"
	tar -C "${ACI_REBUILD_TMP}" -xf "${ACI_IMMD}"
	rm "${ACI_IMMD}"
	tar --mtime="${UPDATE_TIMESTAMP}" -C "${ACI_REBUILD_TMP}" -cf "${ACI_IMMD}.tar" .
	rm -f "${ACI_IMMD}.tar.gz"
	gzip -n "${ACI_IMMD}.tar"
	if [ "${UPDATE_HASH:-}" = "" ]
	then
		echo "Warning: no update hash to check against." 1>&2
	else
		echo "${UPDATE_HASH}  ${ACI_IMMD}.tar.gz" | sha256sum --check
	fi
	mv "${ACI_IMMD}.tar.gz" "${ACI_OUTPUT}"
}

# rkt builder

function build_with_go() {
	if [ ! -e "${GO_TGZ}" ]
	then
		echo "cannot find go binaries" 1>&2
		exit 1
	fi

	if which go
	then
		echo 'should not be any available go executables' 1>&2
		exit 1
	fi

	export GOROOT="${B}/go/"
	tar -C "${B}" -xf "${GO_TGZ}" go/
	export PATH="$PATH:$GOROOT/bin"

	if [ "$(go version 2>/dev/null)" != "go version go'"${GO_VER}"' linux/amd64" ]
	then
		echo 'go version mismatch! expected ${GO_VER}' 1>&2
		go version 1>&2
		exit 1
	fi
}

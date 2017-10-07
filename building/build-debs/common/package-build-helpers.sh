
PKGBASE="$(basename "$(dirname "$(realpath "$0")")")"

if [ "$(echo "${PKGBASE}" | cut -d '-' -f 1)" != "homeworld" ]
then
	echo "Invalid internal name ${PKGBASE} from $0" 1>&2
	exit 1
fi

DVERSION="$(head -n 1 debian/changelog | cut -d '(' -f 2 | cut -d ')' -f 1)"
VERSION="$(echo "$DVERSION" | cut -d '-' -f 1)"
BIN=../binaries
STAGE=..
UPSTREAM=../../upstream
HELPERS="../../build-helpers"
PKGNAME="${PKGBASE}_${DVERSION}_amd64.deb"
PKOUT="${BIN}/${PKGNAME}"
ORIGNAME="${PKGBASE}_${VERSION}.orig.tar.xz"

GOVER=1.8.4
ACVER=0.4.0

GONAME="go-bin-${GOVER}.tgz"
ACBUILDNAME="acbuild-bin-${ACVER}.tgz"

if [ -e "${PKOUT}" ]
then
	echo "${PKGNAME} already built!"
	exit 0
fi

function importgo() {
	if [ ! -e "${HELPERS}/${GONAME}" ]
	then
		echo "No compiled go binary found." 1>&2
		exit 1
	fi
	cp "${HELPERS}/${GONAME}" -T "${GONAME}"
}

function importacbuild() {
	if [ ! -e "${HELPERS}/${ACBUILDNAME}" ]
	then
		echo "No compiled acbuild binary found." 1>&2
		exit 1
	fi
	cp "${HELPERS}/${ACBUILDNAME}" -T "${ACBUILDNAME}"
}

function exportorig() {
	rm -f "${STAGE}/${ORIGNAME}"
	ln -s "${PKGBASE}/${1}" -T "${STAGE}/${ORIGNAME}"
}

function upstream() {
	cp "${UPSTREAM}/${1}" -T "${2:-${1}}"
}

function build() {
	mkdir -p "${BIN}"
	sbuild -d "stretch"
	mv "${STAGE}/${PKGNAME}" -T "${PKOUT}"
}

unset GOROOT

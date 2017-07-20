source ../common/container-build-helpers.sh

common_setup

ROOTFS="${ROOT}/rootfs"
VARIANT="minbase"
COMPONENTS="main"
EXTRA_PACKAGES=""
# MIRROR="http://mirrors.mit.edu/debian/"
# DEBVER=20170719T213259Z

# TODO: hardening by checking release date

function debian_bootstrap() {
	if [ "${RELEASE}" = "" ]
	then
		echo "Release must be specified!" 1>&2
		exit 1
	fi
	if [ "${DEBVER}" = "" ]
	then
		echo "Debian snapshot version must be specified!" 1>&2
		exit 1
	fi
	VERSION="${RELEASE}.${DEBVER}"
	rm -rf "${ROOTFS}"
	mkdir "${ROOTFS}"

	DEBOOTSTRAP_OPTS=(--components="${COMPONENTS}")
	if [ "${VARIANT}" != "" ]
	then
		DEBOOTSTRAP_OPTS+=(--variant="${VARIANT}")
	fi
	if [ "${EXTRA_PACKAGES}" != "" ]
	then
		DEBOOTSTRAP_OPTS+=(--include="${EXTRA_PACKAGES}")
	fi

	debootstrap "${DEBOOTSTRAP_OPTS[@]}" "${RELEASE}" "${ROOTFS}" "http://snapshot.debian.org/archive/debian/${DEBVER}/"
}

function force_remove_packages() {
	if [ "${ROOTFS}" = "" ]
	then
		echo "Failed to get rootfs." 1>&2
		exit 1
	fi
	dpkg --force-remove-essential --root="${ROOTFS}" --purge "$@"
}

function clean_apt_files() {
	rm -rf "${ROOTFS}/var/cache/apt/"
	rm -rf "${ROOTFS}/var/lib/apt/"
	rm -f "${ROOTFS}/var/log/bootstrap.log"
	rm -f "${ROOTFS}/var/log/alternatives.log"
	rm -f "${ROOTFS}/var/log/dpkg.log"
}

function clean_doc_files() {
	rm -rf "${ROOTFS}/usr/share/doc/"
	rm -rf "${ROOTFS}/usr/share/man/"
}

function clean_locales() {
	mkdir "${ROOTFS}/locale/"
	mv "${ROOTFS}"/usr/share/locale/en* -t "${ROOTFS}/locale/"
	rm -rf "${ROOTFS}/usr/share/locale/"
	mv "${ROOTFS}/locale" -t "${ROOTFS}/usr/share/"
}

function clean_ld_aux() {
	rm -f "${ROOTFS}/var/cache/ldconfig/aux-cache"
}

function clean_pycache() {
	# TODO: it might be possible to be smarter than this, since only a small set of files actually vary
	rm -rf "${ROOTFS}/usr/lib/python3.5/unittest/__pycache__"
	rm -rf "${ROOTFS}/usr/lib/python3.5/idlelib/__pycache__"
	rm -rf "${ROOTFS}/usr/lib/python3.5/asyncio/__pycache__"
	rm -rf "${ROOTFS}/usr/lib/python3.5/__pycache__"
}

function write_debian_image() {
	UPDATE_TIMESTAMP="$(echo "${DEBVER}" | sed 's/^\([0-9][0-9][0-9][0-9]\)\([0-9][0-9]\)\([0-9][0-9]\)T\([0-9][0-9]\)\([0-9][0-9]\)\([0-9][0-9]\)Z$/\1-\2-\3 \4:\5:\6 +0000/')"

	start_acbuild "${ROOTFS}"
	$ACBUILD set-working-dir "/"
	$ACBUILD set-exec /bin/bash
	finish_acbuild
}

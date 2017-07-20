source ../common/container-build-helpers.sh

common_setup

ROOTFS="${ROOT}/rootfs"
VARIANT="minbase"
COMPONENTS="main"
EXTRA_PACKAGES=""
MIRROR="http://mirrors.mit.edu/debian/"

function debian_bootstrap() {
	if [ "${RELEASE}" = "" ]
	then
		echo "Release must be specified!" 1>&2
		exit 1
	fi
	VERSION="${RELEASE}.$(date '+%Y.%m.%d.%H')"
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

	debootstrap "${DEBOOTSTRAP_OPTS[@]}" "${RELEASE}" "${ROOTFS}" "${MIRROR}"
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

function write_debian_image() {
	start_acbuild "${ROOTFS}"
	$ACBUILD set-working-dir "/"
	$ACBUILD set-exec /bin/bash
	finish_acbuild
}

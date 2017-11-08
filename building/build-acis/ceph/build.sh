#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

CEPH_VER="12.2.1"
REVISION="3"
VERSION="${CEPH_VER}-${REVISION}"

BUILDACI="ceph-build"
DEBVER="stretch.20171105T160402Z"
BUILDVER="stretch.20171105T160402Z"
UPDATE_TIMESTAMP="2017-11-05T19:27:00-0500"

common_setup

if [ "${TMPDIR:-}" = "" -o "${TMPDIR:-}" = "/tmp" ]
then
	echo 'Warning: $TMPDIR is not set. This may lead to this build failing due to lack of disk space.'
	echo 'Consider launching this script as:'
	echo '    TMPDIR=/home/user/buildtmp/ ./build.sh'
	echo
fi

# build ceph

init_builder

SRCDIR="${BUILDDIR}/src"
DESTDIR="${BUILDDIR}/pkg"
rm -rf "${SRCDIR}" "${DESTDIR}"
mkdir -p "${SRCDIR}" "${DESTDIR}"

tar -C "${BUILDDIR}" -xf "${UPSTREAM}/ceph-${CEPH_VER}.tar.xz" "ceph-${CEPH_VER}/"
mv "${BUILDDIR}/ceph-${CEPH_VER}" -T "${SRCDIR}"

build_at_path "${SRCDIR}"

JOBS="${JOBS:-4}"

function gen_cmake() {
	echo 'export CEPH_BUILD_VIRTUALENV="$(pwd)"'
	echo "mkdir build"
	echo "cd build"
	echo "cmake .. -DCMAKE_INSTALL_PREFIX=/usr -DWITH_MANPAGE=OFF -DWITH_PYTHON3=OFF -DWITH_LTTNG=OFF -DWITH_EMBEDDED=OFF -DWITH_TESTS=OFF -DWITH_CEPHFS=OFF -DBOOST_J=${JOBS} -DWITH_RADOSGW_BEAST_FRONTEND=ON -DWITH_FUSE=OFF"
	echo "make -j${JOBS}"
	echo "make DESTDIR=\"$(path_to_buildpath "${DESTDIR}")\" install"
}
BUILDSCRIPT_GEN+=(gen_cmake)
run_builder

STRIPPABLE="ceph-authtool ceph-bluestore-tool ceph-conf ceph-dencoder ceph-mds ceph-mgr ceph-mon ceph-objectstore-tool ceph-osd ceph-syn rados radosgw radosgw-admin radosgw-es radosgw-object-expirer radosgw-token rbd rbd-mirror rbd-nbd rbd-replay rbd-replay-prep"
for to_strip in ${STRIPPABLE}
do
	strip "${DESTDIR}/usr/bin/${to_strip}"
done

# build container

start_acbuild_from "debian-mini" "${DEBVER}"
$ACBUILD copy-to-dir "${DESTDIR}/usr/bin/"* /usr/bin/
$ACBUILD copy-to-dir "${DESTDIR}/usr/lib/python2.7/dist-packages/"* /usr/lib/python2.7/dist-packages/
$ACBUILD copy-to-dir "${DESTDIR}/usr/lib/x86_64-linux-gnu/"* /usr/lib/x86_64-linux-gnu/
$ACBUILD copy-to-dir "${DESTDIR}/usr/libexec/"* /usr/libexec/
$ACBUILD copy-to-dir "${DESTDIR}/usr/sbin/"* /usr/sbin/
add_packages_to_acbuild cryptsetup-bin debianutils findutils gdisk grep logrotate psmisc xfsprogs btrfs-tools ntp python-cherrypy3 python-openssl python-pecan python-werkzeug python-flask parted python-prettytable python-requests mime-support libibverbs1 libnss3 libaio1 libleveldb1v5 libgoogle-perftools4 libcurl3-gnutls libbabeltrace1
$ACBUILD set-exec -- /bin/bash
finish_acbuild

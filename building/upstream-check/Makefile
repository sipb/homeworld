all: acbuild-src-0.4.0.tgz
all: coreos_binaries-1478.0.0.tar.xz
all: debian-9.6.0-amd64-mini.iso
all: go1.4-bootstrap-20170531.tar.xz
all: go1.9.3.src.tar.xz
all: linux-4.14.16.tar.xz
all: prometheus-client_golang-0.9.0-pre1.tar.xz
all: promu-sipb-0.1.1.tar.xz
PYYAML_VER=3.12
all: PyYAML-${PYYAML_VER}.tar.gz
all: qemu-2.11.0.tar.xz
RKT_VER=1.29.0
all: rkt-${RKT_VER}.tar.xz
all: snapshot.debian.org-20180710T043017Z.stamp

.PHONY: verify just-verify
just-verify:
	sha512sum --check ../SHA512SUM.UPSTREAM

verify: all
	sha512sum --check ../SHA512SUM.UPSTREAM

.PHONY: clean
clean:
	rm -rf *.tar.xz *.tar.gz *.tgz
	rm -rf *.cpio.gz *.cpio *.tar *.iso
	rm -rf *.asc *.sign *.sig

clean: clean-keys
.PHONY: clean-keys
.PRECIOUS: keys/%.ring
clean-keys:
	rm -rf keys/*.ring keys/*.tmp

keys/%.ring: keys/%.asc
	@echo "[gpg] preparing keyring $@"
	@rm -rf $@ $@.tmp
	@mkdir -m=0700 $@.tmp
	@GNUPGHOME="$@.tmp" gpg --quiet --batch --import $^
	@mv "$@.tmp" "$@"

clean: clean-git
.PHONY: clean-git
.PRECIOUS: git/%
clean-git:
	rm -rf git

git:
	@mkdir -p git

git/%: git
	@echo "[git] cloning $(@:git/%=%)"
	@rm -rf "$@"
	@mkdir -p "$(dir $@)"
	@git clone --quiet "https://github.com/$(patsubst git/%,%,$@)" "$@"

TOP := $(shell pwd)

# use as $(call git-verify,<tag>,<repo>,<keyring>)
git-verify = @echo "[git] verifying tag $1 on $2 with key $3" && cd "git/$2" && (GNUPGHOME="${TOP}/keys/$3.ring" git tag -v $1 2>/dev/null >/dev/null || (echo '******* failed to verify' && false))

# we have git-verify-commit for cases where the tagged commit is signed, but not the commit itself
# WARNING: THIS IS MUCH WEAKER, AS SOMEONE COULD SUBSTITUTE A DIFFERENT RELEASE!
# use as $(call git-verify-commit,<tag>,<repo>,<keyring>)
git-verify-commit = @echo "[git] verifying commit for tag $1 on $2 with key $3" && cd "git/$2" && (GNUPGHOME="${TOP}/keys/$3.ring" git verify-commit $1 2>/dev/null >/dev/null || (echo '******* failed to verify' && false))

# use as $(call git-archive,<tag>,<repo>,<prefix>,<output>)
git-archive = @echo "[git] archiving tag $1 on $2" && cd "git/$2" && git archive --format=tar --prefix=$3 --output "${TOP}/$4" $1

clean: clean-http
.PHONY: clean-http
clean-http:
	rm -rf http https

https:
	@mkdir -p https
http:
	@mkdir -p http

http/%: http
	@echo "[http] fetching $(@:http/%=http://%)"
	@mkdir -p $(dir $@)
	@wget --quiet $(@:http/%=http://%) -O $@

https/%: https
	@echo "[https] fetching $(@:https/%=https://%)"
	@mkdir -p $(dir $@)
	@wget --quiet $(@:https/%=https://%) -O $@

# use as $(call gpg-verify,<file>,<signature-suffix>,<keyring>)
gpg-verify = @echo "[gpg] verifying $1 with key $3" && (GNUPGHOME="${TOP}/keys/$3.ring" gpg --quiet --verify $1$2 $1 2>/dev/null >/dev/null || (echo '******* failed to verify' && false))

# use as $(call sha256-verify,<file>,<hash-name>)
sha256-verify = @echo "[sha256] checking $1 against $2" && (echo "$$(cat ${TOP}/known/$2.sha256)  $1" | sha256sum --check --strict --quiet)

%.xz: %
	@rm -f $@
	@echo "[xz] compressing $@"
	@xz -k "$^"

%.tar: %.tar.gz
	@rm -f $@
	@echo "[gzip] decompressing $@"
	@gunzip "$^" --to-stdout >"$@"

%.tgz: %.tar
	@rm -f $@
	@echo "[gzip] compressing $@"
	@gzip -n "$^" --to-stdout >"$@"

# **** ACBUILD ****

acbuild-src-%.tar: git/containers/build keys/acbuild.ring
	$(call git-verify,$(@:acbuild-src-%.tar=v%),containers/build,acbuild)
	$(call git-archive,$(@:acbuild-src-%.tar=v%),containers/build,acbuild/,$@)

# **** RKT ****

rkt-%.tar: git/rkt/rkt keys/coreos-app.ring
	$(call git-verify,$(@:rkt-%.tar=v%),rkt/rkt,coreos-app)
	$(call git-archive,$(@:rkt-%.tar=v%),rkt/rkt,rkt-1.29.0/,$@)

# **** COREOS ****

COREOS_BASE := alpha.release.core-os.net/amd64-usr/%/coreos_production_pxe_image.cpio.gz
coreos_production_pxe_image-%.cpio.gz: https/${COREOS_BASE} https/${COREOS_BASE}.sig keys/coreos-image.ring
	$(call gpg-verify,$<,.sig,coreos-image)
	@cp $< $@

coreos_production_pxe_image-%.cpio: coreos_production_pxe_image-%.cpio.gz
	@echo "[gzip] decompressing $<"
	@gunzip "$<" --to-stdout >"$@"

.PHONY: clean-coreos
clean: clean-coreos
clean-coreos:
	rm -rf rkt-unpacked-* coreos-binary-manifest-* coreos-squashfs-* coreos-unsquash-dir-*

.PRECIOUS: rkt-unpacked-%
rkt-unpacked-%: rkt-%.tar
	@echo "[tar] unpacking working copy of $<"
	@rm -rf $@
	@rm -rf workdir-$@
	@mkdir workdir-$@
	@tar -C workdir-$@ -xf $<
	@mv workdir-$@/$(@:rkt-unpacked-%=rkt-%) $@
	@rm -rf workdir-$@

coreos-binary-manifest-%.txt: rkt-unpacked-%
	@echo "[coreos] compiling rkt manifest from $<"
	@find $< -name '*.manifest' -print0 | grep -z amd64 | xargs -0 cat -- | sort -u >$@

coreos-squashfs-%: coreos_production_pxe_image-%.cpio
	@echo "[cpio] extracting usr.squashfs from $<"
	@cpio --quiet --extract --to-stdout usr.squashfs <$< >$@

.PRECIOUS: coreos-unsquash-dir-%
coreos-unsquash-dir-%: coreos-squashfs-% coreos-binary-manifest-${RKT_VER}.txt
	@echo "[coreos] unsquashing $@ with manifest $(word 2,$^)"
	@rm -rf $@
	@unsquashfs -no-progress -no-xattrs -d $@ -e $(word 2,$^) $< >/dev/null

coreos_binaries-%.tar: coreos-unsquash-dir-%
	@echo "[tar] packing $@"
	@rm -rf workdir-$@ $@ $@.tmp
	@mkdir workdir-$@
	@mv $^ "workdir-$@/coreos_minimal_dir/"
	@tar --sort=name --owner=user:1000 --group=user:1000 -C workdir-$@ --mtime="Mon, 1 Jan 2018 00:00:00 -0500" -cf $@.tmp "coreos_minimal_dir/"
	@mv $@.tmp $@

# **** DEBIAN ****

clean: clean-debian-stretch-release
.PHONY: clean-debian-stretch-release
clean-debian-stretch-release:
	rm -f debian-stretch-release

debian-stretch-release: http/debian.csail.mit.edu/debian/dists/stretch/Release http/debian.csail.mit.edu/debian/dists/stretch/Release.gpg keys/debian.ring
	$(call gpg-verify,$<,.gpg,debian)
	@cp http/debian.csail.mit.edu/debian/dists/stretch/Release $@

debian-9.6.0-amd64-mini.iso: debian-20170615+deb9u5-amd64-mini.iso
	@cp $< $@

debian-%-amd64-mini.iso: http/debian.csail.mit.edu/debian/dists/stretch/main/installer-amd64/%/images/netboot/mini.iso \
                         http/debian.csail.mit.edu/debian/dists/stretch/main/installer-amd64/%/images/SHA256SUMS \
                         debian-stretch-release
	@rm -rf "workdir-$@" "$@"
	@mkdir -p workdir-$@/netboot
	@cp $(word 2,$^) workdir-$@/SHA256SUMS
	@echo "[gpg] verifying hash for installer SHA256SUMS against release"
	@echo "$$(grep $(patsubst http/debian.csail.mit.edu/debian/dists/stretch/%,%,$(word 2,$^)) debian-stretch-release | tail -n 1 | cut -d " " -f 2)  workdir-$@/SHA256SUMS" | sha256sum --check --strict --quiet
	@cp $< workdir-$@/netboot/mini.iso
	@echo "[gpg] verifying hash for iso against installer SHA256SUMS"
	@cd workdir-$@ && grep ./netboot/mini.iso SHA256SUMS | sha256sum --check --strict --quiet
	@cp workdir-$@/netboot/mini.iso $@

# **** GOLANG ****

go1.4-bootstrap-%.tar.gz: https/storage.googleapis.com/golang/go1.4-bootstrap-%.tar.gz
	@rm -f $@ $@.tmp
	@cp $< $@.tmp
	$(call sha256-verify,$@.tmp,go-bootstrap)
	@mv $@.tmp $@

go%.src.tar.gz: https/dl.google.com/go/go%.src.tar.gz
	@rm -f $@ $@.tmp
	@cp $< $@.tmp
	$(call sha256-verify,$@.tmp,go-1.9.3)
	@mv $@.tmp $@

# **** LINUX ****

linux-4.%.tar.xz: https/www.kernel.org/pub/linux/kernel/v4.x/linux-4.%.tar.xz https/www.kernel.org/pub/linux/kernel/v4.x/linux-4.%.tar.sign keys/linux.ring
	@cp $< $(@:.tar.xz=.tmp.tar.xz)
	@echo "[xz] extracting $@"
	@unxz $(@:.tar.xz=.tmp.tar.xz)
	@cp $(word 2,$^) $(@:.tar.xz=.tmp.tar.sign)
	$(call gpg-verify,$(@:.tar.xz=.tmp.tar),.sign,linux)
	@cp $< $@

# **** PROMETHEUS-CLIENT_GOLANG ****

prometheus-client_golang-%.tar: git/prometheus/client_golang
	@rm -f $@ $@.tmp
	$(call git-archive,$(@:prometheus-client_golang-%.tar=v%),prometheus/client_golang,$(@:prometheus-%.tar=%/),$@.tmp)
	$(call sha256-verify,$@.tmp,$(@:%.tar=%))
	@mv $@.tmp $@

# **** PROMU-SIPB ****

promu-sipb-%.tar: git/sipb/promu keys/promu.ring
	$(call git-verify-commit,$(@:promu-%.tar=%),sipb/promu,promu)
	$(call git-archive,$(@:promu-%.tar=%),sipb/promu,$(@:%.tar=%/),$@)

# **** PYYAML ****

# I don't believe that this will handle incrementing versions correctly. When that happens, though, we should switch to grabbing these from the git repo.
PyYAML-${PYYAML_VER}.tar.gz: https/pypi.python.org/packages/4a/85/db5a2df477072b2902b0eb892feb37d88ac635d36245a72a6a69b23b383a/PyYAML-${PYYAML_VER}.tar.gz
	$(call sha256-verify,$<,$(@:%.tar.gz=%))
	@cp $< $@

# **** QEMU ****

qemu-%.tar.xz: https/download.qemu.org/qemu-%.tar.xz https/download.qemu.org/qemu-%.tar.xz.sig keys/qemu.ring
	$(call gpg-verify,$<,.sig,qemu)
	@cp $< $@

# **** SNAPSHOT.DEBIAN.ORG ****

snapshot.debian.org-%.stamp:
	@echo "[snapshot] mirroring snapshot.debian.org version $(@:snapshot.debian.org-%.stamp=%)"
	@python3 snapshot-pull.py "$(@:snapshot.debian.org-%.stamp=%)" <packages.list
	@touch $@

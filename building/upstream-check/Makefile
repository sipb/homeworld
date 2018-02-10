all: acbuild-src-0.4.0.tgz
all: coreos_binaries-1478.0.0.tar.xz
all: debian-9.2.0-amd64-mini.iso
all: dnsmasq-2.78.tar.xz
all: etcd-3.2.9.tar.xz
all: flannel-0.10.0.tar.xz
all: go1.4-bootstrap-20170531.tar.xz
all: go1.9.3.src.tar.xz
all: golang-x-crypto-5ef0053f77724838734b6945dd364d3847e5de1d.tar.xz
all: gopkg.in-yaml.v2-eb3733d160e74a9c7e442f435eb3bea458e1d19f.tar.xz
all: knc-1.7.1.tar.xz
all: kubernetes-dns-1.14.8.tar.xz
all: kubernetes-src-v1.9.2.tar.xz
all: kube-state-metrics-1.2.0.tar.xz
all: linux-4.14.16.tar.xz
all: prometheus-2.0.0.tar.xz
all: prometheus-client_golang-0.9.0-pre1.tar.xz
all: prometheus-node-exporter-0.15.2.tar.xz
all: promu-sipb-0.1.1.tar.xz
all: PyYAML-3.12.tar.gz
all: qemu-2.11.0.tar.xz
RKT_VER=1.29.0
all: rkt-${RKT_VER}.tar.xz

.PHONY: clean
clean:
	rm -rf *.tar.xz
	rm -rf *.tar.gz
	rm -rf *.tgz
	rm -rf *.tar
	rm -rf *.iso

clean: clean-keys
.PHONY: clean-keys
.PRECIOUS: keys/%.ring
clean-keys:
	rm -rf keys/*.ring keys/*.tmp

keys/%.ring: keys/%.asc
	@echo "[gpg] preparing keyring $@"
	@rm -rf $@ $@.tmp
	@mkdir -m=0700 $@.tmp
	@GNUPGHOME="$@.tmp" gpg --quiet --import $^
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
	@git clone --quiet "https://github.com/$(patsubst git/%,%,$@)/" "$@"

TOP := $(shell pwd)

# use as $(call git-verify,<tag>,<repo>,<keyring>)
git-verify = @echo "[git] verifying tag $1 on $2 with key $3" && cd "git/$2" && (GNUPGHOME="${TOP}/keys/$3.ring" git tag -v $1 2>/dev/null >/dev/null || (echo '******* failed to verify' && false))
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

%.xz: %
	@echo "[xz] compressing $@"
	@xz "$^"

%.tgz: %.tar
	@echo "[gzip] compressing $@"
	@gzip "$^" --to-stdout >"$@"

# **** ACBUILD ****

acbuild-src-%.tar: git/containers/build keys/acbuild.ring
	# TODO: ensure that the correct signature is checked properly
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
	@mv $< $@

coreos_production_pxe_image-%.cpio: coreos_production_pxe_image-%.cpio.gz
	@echo "[gzip] uncompressing $<"
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
	@tar -C workdir-$@ --mtime="2018-01-01 12:00:00 AM" -cf $@.tmp "coreos_minimal_dir/"
	@mv $@.tmp $@

# **** DEBIAN ****

clean: clean-debian-stretch-release
.PHONY: clean-debian-stretch-release
clean-debian-stretch-release:
	rm -f debian-stretch-release

debian-stretch-release: http/debian.csail.mit.edu/debian/dists/stretch/Release http/debian.csail.mit.edu/debian/dists/stretch/Release.gpg keys/debian.ring
	$(call gpg-verify,$<,.gpg,debian)
	@cp http/debian.csail.mit.edu/debian/dists/stretch/Release $@

debian-9.2.0-amd64-mini.iso: debian-20170615+deb9u2+b1-amd64-mini.iso
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

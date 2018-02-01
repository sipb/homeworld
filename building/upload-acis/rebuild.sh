#!/bin/bash

set -e -u

cd "$(dirname "$0")"
mkdir -p staging

cp ../build-acis/containers/*.aci -t staging
for x in staging/*.aci
do
	if [ ! -e "$x.asc" ] && [ -f "$x" ]
	then
		gpg --armor --detach-sign --local-user 0x8422464D9EE78588 "$x"
	fi
done

function find_latest() {
	LATEST="$(ls "staging/$1"*-linux-amd64.aci | grep -vF latest | sort | tail -n 1)"
	echo "Latest for $1: $LATEST"
	ln -sf "$(basename "$LATEST")" "staging/${1}latest-linux-amd64.aci"
	ln -sf "$(basename "$LATEST.asc")" "staging/${1}latest-linux-amd64.aci.asc"
	if [ "${2:-}" != "" ]
	then
		ln -sf "$(basename "$LATEST")" "staging/${2}-linux-amd64.aci"
		ln -sf "$(basename "$LATEST.asc")" "staging/${2}-linux-amd64.aci.asc"
	fi
}

for x in debian-build debian debian-mini debian-micro
do
	find_latest "$x-stretch." "$x-latest"
done

find_latest "flannel-0.10.0-" "flannel-latest"
find_latest "kube-state-metrics-1.2.0-" "kube-state-metrics-latest"
find_latest "flannel-monitor-"
find_latest "pullcheck-"
find_latest "dns-monitor-"
find_latest "kube-dns-main-1.14.8-" "kube-dns-main-latest"
find_latest "kube-dns-sidecar-1.14.8-" "kube-dns-sidecar-latest"
find_latest "dnsmasq-2.78-" "dnsmasq-latest"
find_latest "dnsmasq-nanny-1.14.8-" "dnsmasq-nanny-latest"

sleep 0.1

DEST=/mit/hyades/acis/homeworld.mit.edu/
for x in staging/*.aci staging/*.aci.asc
do
	FILENAME="$(basename "$x")"
	echo "checking $x"
	NEEDS_COPY=false
	if [ ! -e "${DEST}/${FILENAME}" ]
	then
		NEEDS_COPY=true
	elif [ "$(wc -c < "${x}")" != "$(wc -c < "${DEST}/${FILENAME}")" ]
	then
		NEEDS_COPY=true
	elif [ "${x%.asc}" != "${x}" ]
        then
		if cmp --silent "${x}" "${DEST}/${FILENAME}"
		then
			true  # do nothing
		else
			NEEDS_COPY=true
		fi
	fi
	if "${NEEDS_COPY}"
	then
		echo "copying $x"
		cp -dfT "$x" "${DEST}/${FILENAME}"
	fi
done
echo "done!"

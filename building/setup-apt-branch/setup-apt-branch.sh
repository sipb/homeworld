if [ -z "${HOMEWORLD_APT_BRANCH:-}" ]
then
	echo 'Error: Need to specify apt branch:' >&2
	echo '$ export HOMEWORLD_APT_BRANCH=<username>/<branch>' >&2
	echo 'Use root/master if you would like to base this off the main repository.' >&2
	exit 1
fi

if ! [[ "${HOMEWORLD_APT_BRANCH}" =~ ^[0-9a-zA-Z_-]+\/[0-9a-zA-Z_-]+$ ]]
then
	echo 'Error: Apt branch invalid. Should be of the form <username>/<branch>.' >&2
	echo 'Allowed characters: [0-9a-zA-Z_-]' >&2
	exit 1
fi

function get_apt_signing_key()
{
    while read branch key; do
        if [ "${branch}" == "${HOMEWORLD_APT_BRANCH}" ] || [ "${branch}" == "$(echo ${HOMEWORLD_APT_BRANCH} | cut -d '/' -f 1)" ]; then
            HOMEWORLD_APT_SIGNING_KEY="${key}"
            break
        fi
    done < `dirname $BASH_SOURCE`/signing-keys

    if [ -z "${HOMEWORLD_APT_SIGNING_KEY:-}" ]
    then
        echo 'error: apt branch not found in signing-keys' >&2
        exit 1
    fi

    if ! [[ "${HOMEWORLD_APT_SIGNING_KEY}" =~ ^[0-9a-zA-Z]+$ ]]
    then
        echo 'error: apt signing key invalid' >&2
        exit 1
    fi

    gpg --list-keys "${HOMEWORLD_APT_SIGNING_KEY}" > /dev/null
    if [ $? -ne 0 ]
    then
        echo 'error: apt signing key not in gpg keyring' >&2
        exit 1
    fi

    echo "${HOMEWORLD_APT_SIGNING_KEY}"
}

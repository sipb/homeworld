#!/bin/bash
set -e -u -o pipefail
TOKEN="${1%??}"
TOKEN_PROVIDED_HASH=$(echo -n $1 | tail -c 2)

TOKEN_ACTUAL_HASH=$(echo -n "$TOKEN" | /usr/bin/openssl dgst -sha256 -binary | /usr/bin/base64 | cut -c -2)

[ "$TOKEN_PROVIDED_HASH" == "$TOKEN_ACTUAL_HASH" ]

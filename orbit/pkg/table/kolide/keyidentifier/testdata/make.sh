#!/bin/bash

# This script uses several tools to create ssh keys. We attempt to be
# as exhaustive as possible to create a wide range of things to test.
#
# each function also generates a json spec file that describes various
# attributes of a key, plus the name of the key described. This spec file is
# consumed by golang tests, and the output of the keyidentifier package is
# compared against these expected values.

set -e
set -o pipefail
# set -x

function rand {
    set +o pipefail
    LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 16
    set -o pipefail
}

DATA_DIR="specs"

function makeOpensshKeyAndSpec {
    type=$1
    bits=$2
    encrypted=$3
    format="openssh7"
    source="openssh"

    keypath="$DATA_DIR/$(rand)"

    comment="comment -- $type/$bits by $source $(rand)"

    # set a bash array to the command, then use parameter expansion to
    # invoke it. Using an array, lets us ensure consistency between
    # what we call, and what we record.
    cmd=(ssh-keygen -t $type -b $bits -f $keypath -C "$comment")

    if [ $encrypted == true ]; then
        cmd+=(-P "$(rand)")
    else
        cmd+=(-P "")
    fi

    echo "${cmd[@]}"
    "${cmd[@]}"
    #echo returned $?


    fingerprint=$(ssh-keygen -l -f $keypath.pub | awk '{print $2}' | sed -e 's/^SHA256://')
    md5fingerprint=$(ssh-keygen -l -E md5 -f $keypath.pub | awk '{print $2}' | sed -e 's/^MD5://')

    cat <<EOF > $keypath.json
{
    "Comment": "$comment",
    "FingerprintSHA256": "$fingerprint",
    "FingerprintMD5": "$md5fingerprint",
    "Type": "$type",
    "Bits": $bits,
    "Encrypted": $encrypted,
    "command": "${cmd[@]}",
    "Format": "openssh-new",
    "Source": "ssh-keygen"
}
EOF
}

# ssh.com style
#
# Note that openssh's ssh-keygen proportes to convert (using `-e`) but
# empirically this does not work. So we use puttygen (`brew install
# putty` to generate these)
function makePuttyKeyAndSpecFile {
    type=$1
    bits=$2
    format=$3
    encrypted=$4
    source="putty"

    comment="comment -- $type/$bits by $source $(rand)"

    putty_format="-$format"
    if [ "$format" == "putty" ]; then
        putty_format=""
        if [ "$type" == "rsa1" ]; then
            format="ssh1"
        fi
    fi

    keypath="$DATA_DIR/$(rand)"

    # We use `echo ""` instead of /dev/null, because it makes it
    # easier to be consistent in how we extract the public key
    passphrase=""
    if [ $encrypted == true ]; then
        passphrase=$(rand)
    fi

    cmd=(puttygen --random-device /dev/urandom -t $type -b $bits -o $keypath -O private$putty_format --new-passphrase <(echo -n "$passphrase") -C "$comment")

    echo "${cmd[@]}"
    "${cmd[@]}"
    #echo returned $?


    # make the public key pair
    puttygen -L --old-passphrase <(echo -n "$passphrase") $keypath > $keypath.pub

    # puttygen does not directly support the sha256 fingerprint. But, we can use ssh-keygen for it.
    # Usually. ssh-key fails on very old keys. So we probably just need to shrug about those.
    md5fingerprint=$(puttygen -l $keypath --old-passphrase <(echo -n $passphrase) | awk '{print $3}')
    fingerprint=""

    # TODO: figure out how to get the md5 fingerprint for these
    if [ "$type" != "rsa1" ]; then
        fingerprint=$(ssh-keygen -l -f $keypath.pub | awk '{print $2}' | sed -e 's/^SHA256://')
    fi

    cat <<EOF > $keypath.json
{
    "Comment": "$comment",
    "FingerprintSHA256": "$fingerprint",
    "FingerprintMD5": "$md5fingerprint",
    "Type": "$type",
    "Bits": $bits,
    "Encrypted": $encrypted,
    "command": "$(echo -n ${cmd[*]})",
    "Format": "$format",
    "Source": "$source"
}
EOF
}

# Prep

# check if puttygen is installed
hash puttygen 2>/dev/null || \
    { echo >&2 "puttygen must be installed to generate test data for putty keys. use 'brew install putty' to install puttygen on macos"; exit 1; }

mkdir -p "$DATA_DIR"


# -------------------------------------------------------------------------------------------
# Actually make all the keys now that the functions have been defined
# -------------------------------------------------------------------------------------------



makeOpensshKeyAndSpec rsa 1024 true
makeOpensshKeyAndSpec rsa 1024 false

makeOpensshKeyAndSpec rsa 2048 true
makeOpensshKeyAndSpec rsa 2048 false

makeOpensshKeyAndSpec rsa 4096 true
makeOpensshKeyAndSpec rsa 4096 false

makeOpensshKeyAndSpec dsa 1024 true
makeOpensshKeyAndSpec dsa 1024 false

makeOpensshKeyAndSpec ecdsa 256 true
makeOpensshKeyAndSpec ecdsa 256 false

makeOpensshKeyAndSpec ecdsa 521 true
makeOpensshKeyAndSpec ecdsa 521 false


# rsa1 is only supported in the old old old format
makePuttyKeyAndSpecFile rsa1 1024 putty true
makePuttyKeyAndSpecFile rsa1 1024 putty false

makePuttyKeyAndSpecFile rsa1 2048 putty true
makePuttyKeyAndSpecFile rsa1 2048 putty false

for key_format in openssh openssh-new sshcom putty; do
    makePuttyKeyAndSpecFile rsa 1024 $key_format true
    makePuttyKeyAndSpecFile rsa 1024 $key_format false

    makePuttyKeyAndSpecFile rsa 2048 $key_format true
    makePuttyKeyAndSpecFile rsa 2048 $key_format false

    makePuttyKeyAndSpecFile rsa 4096 $key_format true
    makePuttyKeyAndSpecFile rsa 4096 $key_format false

    makePuttyKeyAndSpecFile dsa 1024 $key_format true
    makePuttyKeyAndSpecFile dsa 1024 $key_format false
done

for key_format in openssh openssh-new putty; do

    makePuttyKeyAndSpecFile ecdsa 256 $key_format true
    makePuttyKeyAndSpecFile ecdsa 256 $key_format false

    makePuttyKeyAndSpecFile ecdsa 521 $key_format true
    makePuttyKeyAndSpecFile ecdsa 521 $key_format false

    makePuttyKeyAndSpecFile ed25519 256 $key_format true
    makePuttyKeyAndSpecFile ed25519 256 $key_format false
done

# openssl genpkey -algorithm RSA -out private_key.pem -pkeyopt rsa_keygen_bits:2048
# openssl genpkey -algorithm RSA -pass pass:password -out private_key_enc.pem -pkeyopt rsa_keygen_bits:2048

echo "done"

#!/bin/bash

#
# Test script to setup a local Munki repository for demo/testing purposes.
# Sets latest Firefox dmg on a client manifest.
#

if [[ -z "$REPO_DIR" ]]; then
    echo "Set REPO_DIR to an absolute file path."
    exit 1
fi

if [[ $REPO_DIR != /* ]]; then
    echo "REPO_DIR must be an absolute file path."
    exit 1
fi

if [[ -d "$REPO_DIR" ]]; then
    echo -n "REPO_DIR=$REPO_DIR already exists, press any key to delete and continue... "
    read
    rm -rf $REPO_DIR
fi

mkdir -p $REPO_DIR/catalogs
mkdir $REPO_DIR/icons
mkdir $REPO_DIR/manifests
mkdir $REPO_DIR/pkgs
mkdir $REPO_DIR/pkgsinfo

curl -L "https://download.mozilla.org/?product=firefox-latest-ssl&os=osx&lang=en-US" --output $(pwd)/firefox.dmg

/usr/local/munki/munkiimport \
    --nointeractive \
    --repo_url=file://$REPO_DIR \
    --subdirectory=apps/mozilla \
    --displayname="Mozilla Firefox" \
    --description="Fox on fire" \
    --category=Internet \
    --developer=Mozilla \
    --catalog=testing \
    --extract_icon \
    $(pwd)/firefox.dmg
/usr/local/munki/makecatalogs --repo_url=file://$REPO_DIR

# No other (non-interactive) way to set the repo url for manifestutil.
defaults write ~/Library/Preferences/com.googlecode.munki.munkiimport.plist "repo_url" "file://$REPO_DIR"
defaults write ~/Library/Preferences/com.googlecode.munki.munkiimport.plist "default_catalog" "testing"

/usr/local/munki/manifestutil new-manifest site_default
/usr/local/munki/manifestutil add-catalog testing --manifest site_default
/usr/local/munki/manifestutil add-pkg Firefox --manifest site_default

rm $(pwd)/firefox.dmg


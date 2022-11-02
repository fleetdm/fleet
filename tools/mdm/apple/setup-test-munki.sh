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

curl -L "https://download.mozilla.org/?product=firefox-latest-ssl&os=osx&lang=en-US" --output firefox.dmg
curl -L "https://app-updates.agilebits.com/download/OPM7" --output 1password7.pkg
curl -L "https://github.com/macadmins/nudge/releases/download/v1.1.8.81422/Nudge-1.1.8.81422.pkg" --output nudge.pkg
curl -L "https://iterm2.com/downloads/stable/iTerm2-3_4_16.zip" --output iterm2.zip
unzip iterm2.zip
rm iterm2.zip
curl -L "https://central.github.com/deployments/desktop/desktop/latest/darwin" --output github.zip
unzip github.zip
rm github.zip

# No other (non-interactive) way to set the repo url for manifestutil.
defaults write ~/Library/Preferences/com.googlecode.munki.munkiimport.plist "repo_url" "file://$REPO_DIR"
defaults write ~/Library/Preferences/com.googlecode.munki.munkiimport.plist "default_catalog" "testing"

# Add Firefox with "--unattended_install" (dmg).
/usr/local/munki/munkiimport \
    --nointeractive \
    --subdirectory=apps/mozilla \
    --displayname="Mozilla Firefox" \
    --description="Fox on fire" \
    --category=Internet \
    --developer=Mozilla \
    --catalog=testing \
    --extract_icon \
    --unattended_install \
    firefox.dmg

# Add 1Password (pkg).
/usr/local/munki/munkiimport \
    --nointeractive \
    --subdirectory=apps/agilebits \
    --displayname="1Password 7" \
    --description="P4ssw0rd M4n4g3r" \
    --category=Internet \
    --developer=AgileBits \
    --catalog=testing \
    --extract_icon \
    1password7.pkg

# Add Nudge with "--unattended_install" (pkg).
/usr/local/munki/munkiimport \
    --nointeractive \
    --subdirectory=apps/macadmins \
    --displayname="Nudge" \
    --description="Annoying but effective" \
    --category=Internet \
    --developer=MacAdmins \
    --catalog=testing \
    --extract_icon \
    --unattended_install \
    nudge.pkg

# Add iTerm2 app.
/usr/local/munki/munkiimport \
    --nointeractive \
    --subdirectory=apps/iterm2 \
    --displayname="iTerm2" \
    --description="Best terminal in town" \
    --category=Console \
    --developer=iTerm2 \
    --catalog=testing \
    --extract_icon \
    iTerm.app

# Add Github app.
/usr/local/munki/munkiimport \
    --nointeractive \
    --subdirectory=apps/github \
    --displayname="Github Desktop" \
    --description="Github 4 Desktop" \
    --category=Development \
    --developer=Github \
    --catalog=testing \
    --extract_icon \
    "Github Desktop.app"

/usr/local/munki/makecatalogs

/usr/local/munki/manifestutil new-manifest site_default
/usr/local/munki/manifestutil add-catalog testing --manifest site_default

/usr/local/munki/manifestutil add-pkg Firefox --manifest site_default
/usr/local/munki/manifestutil add-pkg 1password --manifest site_default
/usr/local/munki/manifestutil add-pkg nudge --manifest site_default
/usr/local/munki/manifestutil add-pkg iTerm2 --manifest site_default --section optional_installs
/usr/local/munki/manifestutil add-pkg "GitHub Desktop" --manifest site_default --section featured_items
/usr/local/munki/manifestutil add-pkg "GitHub Desktop" --manifest site_default --section optional_installs

rm -r firefox.dmg nudge.pkg 1password7.pkg iTerm.app "Github Desktop.app"


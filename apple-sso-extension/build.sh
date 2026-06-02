#!/bin/bash
# Build, sign, package, and notarize Fleet PSSO.
#
# Output: ./FleetPSSO.pkg — a Developer ID-signed, notarized installer that
# drops FleetPSSO.app into /Applications and brings the bundled SSO appex
# along for the ride.
#
# Required env for notarization:
#   AC_USERNAME  Apple ID email
#   AC_TEAM_ID   Apple Developer Team ID
#   AC_PASSWORD  App-specific password (use @keychain:... for keychain refs)

set -euo pipefail

# Not checked in
APP_PROFILE="./profiles/PSSO_Testing_App.provisionprofile"
APPEX_PROFILE="./profiles/PSSO_Testing_Extension.provisionprofile"

# TODO Update
TEAM_ID='5K28R5ZUK5'
NAME_WITH_TEAM="Elijah Montgomery (${TEAM_ID})"
APP_SIGN_ID="Developer ID Application: ${NAME_WITH_TEAM}"
PKG_SIGN_ID="Developer ID Installer: ${NAME_WITH_TEAM}"

APP_PATH="./build/Build/Products/Release/FleetPSSO.app"
APPEX_PATH="${APP_PATH}/Contents/PlugIns/FleetPSSOExtension.appex"
PKG_PATH="./FleetPSSO.pkg"

# Developer ID provisioning profiles. Required to make codesign honor the
# `com.apple.developer.*` entitlements (associated-domains, etc.). Download
# these from the Apple Developer portal under Profiles → Developer ID for
# each App ID and place them at the paths below.

for p in "${APP_PROFILE}" "${APPEX_PROFILE}"; do
  if [[ ! -f "${p}" ]]; then
    echo "ERROR: missing provisioning profile ${p}" >&2
    echo "  Download Developer ID profiles for the host app and extension" >&2
    echo "  from https://developer.apple.com/account/resources/profiles/list" >&2
    exit 1
  fi
done

echo "==> Clean build (unsigned; we re-sign with Developer ID below)"
xcodebuild -project "Fleet PSSO.xcodeproj" -scheme FleetPSSO -configuration Release -derivedDataPath ./build \
  CODE_SIGN_IDENTITY="" \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGNING_ALLOWED=NO \
  clean build

echo "==> Embedding extension provisioning profile"
cp "${APPEX_PROFILE}" "${APPEX_PATH}/Contents/embedded.provisionprofile"

echo "==> Signing SSO extension"
codesign --force --options runtime --timestamp \
  --sign "${APP_SIGN_ID}" \
  --entitlements FleetPSSOExtension/FleetPSSOExtension.entitlements \
  "${APPEX_PATH}"

echo "==> Embedding host app provisioning profile"
cp "${APP_PROFILE}" "${APP_PATH}/Contents/embedded.provisionprofile"

echo "==> Signing host app"
codesign --force --options runtime --timestamp \
  --sign "${APP_SIGN_ID}" \
  --entitlements FleetPSSO/FleetPSSO.entitlements \
  "${APP_PATH}"

echo "==> Building installer pkg (installs to /Applications)"
pkgbuild \
  --component "${APP_PATH}" \
  --install-location /Applications \
  --sign "${PKG_SIGN_ID}" \
  --timestamp \
  "${PKG_PATH}"

echo "==> Notarizing pkg"
xcrun notarytool submit "${PKG_PATH}" \
  --apple-id "${AC_USERNAME}" \
  --team-id "${AC_TEAM_ID}" \
  --password "${AC_PASSWORD}" \
  --wait

echo "==> Stapling notarization ticket to pkg"
xcrun stapler staple "${PKG_PATH}"

echo ""
echo "Done. Install with:"
echo "  sudo installer -pkg ${PKG_PATH} -target /"

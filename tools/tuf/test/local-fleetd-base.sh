#!/bin/sh



./build/fleetctl package --type pkg --fleet-desktop --use-system-configuration --sign-identity $SIGNING_IDENTITY --update-url=$LOCAL_TUF_URL --debug --update-roots=$UPDATE_ROOTS

echo "<plist version="1.0">
  <dict>
    <key>items</key>
    <array>
      <dict>
        <key>assets</key>
        <array>
          <dict>
            <key>kind</key>
            <string>software-package</string>
            <key>sha256-size</key>
            <integer>32</integer>
            <key>sha256s</key>
            <array>
            <string>$(shasum -a 256 fleet-osquery.pkg | awk '{ print $1 }')</string>
            </array>
            <key>url</key>
            <string>https://jve-images-snicket.ngrok.app/fleet-osquery.pkg</string>
          </dict>
        </array>
      </dict>
    </array>
  </dict>
</plist>" >/Users/jahziel/source/fleetdm/tmp/imageserver/stable/fleetd-base-manifest.plist

echo "updated manifest"
rm /Users/jahziel/source/fleetdm/tmp/imageserver/fleet-osquery.pkg
echo "removed old fleetd base"
cp ./fleet-osquery.pkg /Users/jahziel/source/fleetdm/tmp/imageserver
echo "moved old fleetd base to server dir"

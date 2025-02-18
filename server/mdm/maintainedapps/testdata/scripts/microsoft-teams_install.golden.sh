#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# install pkg files

CHOICE_XML=$(mktemp /tmp/choice_xml_XXX)

cat << EOF > "$CHOICE_XML"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <array>
    <dict>
      <key>attributeSetting</key>
      <integer>0</integer>
      <key>choiceAttribute</key>
      <string>selected</string>
      <key>choiceIdentifier</key>
      <string>com.microsoft.autoupdate</string>
    </dict>
  </array>
</plist>

EOF

sudo installer -pkg "$TMPDIR"/MicrosoftTeams.pkg -target / -applyChoiceChangesXML "$CHOICE_XML"


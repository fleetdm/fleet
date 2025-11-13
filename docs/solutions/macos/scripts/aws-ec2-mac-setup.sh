#!/bin/bash

# This script is to set up an AWS EC2 Mac instance with auto-login for MDM enrollment.
# Paste this script into the "User data" field when launching your EC2 Mac instance. 
# Once launched, connect via VNC or Apple Screen Sharing to accept the permissions prompts.
# Once the prompts are accepted, after clicking OK, create the image:
# In the AWS console, go to EC2, click the instance in the list, then click Image and templates ->  Create image.
# Launching new instances from this AMI will automatically enroll them into MDM.

# Set the below to your AWS Secrets Manager secret containing credentials.
# Credential block template available at enroll-ec2-mac repo. This secret name will also be written to the image for enrollment as part of this script.
credentialID="SET-TO-YOUR-SECRETS-MANAGER-SECRET-NAME"

# Path prefix for AWS CLI.
PATH="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/opt/homebrew/bin:/opt/homebrew/sbin"

# JSONVoorhees: made from code by Matthew Warren, under MIT license.
# https://macblog.org/parse-json-command-line-mac/
function jsonParse() (
JSONVar="${1}"
INCOMINGDATA="${2}"
JSONReturn=$( osascript -l 'JavaScript' <<< "function run() { var jsoninfo = JSON.parse(\`$INCOMINGDATA\`) ; return jsoninfo.$JSONVar;}" )
echo "${JSONReturn}"
)

# The below retry function is used for AWS Secrets Manager retrieval only.
function retry() {
local n max waitFor
n=1
max=5
waitFor=15
while true; do
"${@}" && break || {
    if [[ "${n}" -lt "${max}" ]]; then
    ((n++))
    echo "Command failed. Attempt ${n}/${max}:" >&2; 
    sleep "${waitFor}";
    else
    echo "The command has failed after ${n} attempts." ; exit 111
    fi
}
done
}

### Metadata token for authorization to retrieve data from local EC2 Mac instance,
MDToken=$(curl -X PUT "http://169.254.169.254/latest/api/token" -s -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")

### Get current AWS region from running instance metadata.
### If Secrets Manager secret is in a different region, uncomment below to set a static region.
currentRegion=$(curl -H "X-aws-ec2-metadata-token: $MDToken" -s http://169.254.169.254/latest/meta-data/placement/region)
# currentRegion="us-east-1"

# Retrieves credentials from AWS Secrets Manager. To set statically, comment out credentialBlock and set EBSAdminUser and EBSAdminPassword.
credentialBlock=$(retry aws secretsmanager get-secret-value --region "$currentRegion" --secret-id "$credentialID" --query SecretString --output text)

# EBSAdminUser is "ec2-user" by default. If another account is named, it will be created (as an administrator).
EBSAdminUser=$(jsonParse "localAdmin" "$credentialBlock")
EBSAdminPassword=$(jsonParse "localAdminPassword" "$credentialBlock")

function createUserAccount () (
	userToCreate="${1}"
	userPassword="${2}"
	userID="${3}"
    sudo /usr/sbin/sysadminctl -addUser "$userToCreate" -fullName "$userToCreate" -UID "$userID" -GID 80 -shell /bin/zsh -password "$userPassword" -home "/Users/$userToCreate"
    sudo /usr/sbin/createhomedir -c -u "$userToCreate"
)

function kcpasswordEncode () (
	thisString="${1}"
	cipherHex_array=( 7D 89 52 23 D2 BC DD EA A3 B9 1F )
	thisStringHex_array=( $(/bin/echo -n "${thisString}" | xxd -p -u | sed 's/../& /g') )
	if [ "${#thisStringHex_array[@]}" -lt 12  ]; then
		padding=$(( 12 -  ${#thisStringHex_array[@]} ))
	elif [ "$(( ${#thisStringHex_array[@]} % 12 ))" -ne 0  ]; then
		padding=$(( (12 - ${#thisStringHex_array[@]} % 12) ))
	else
		padding=12
	fi	
	for ((i=0; i < $(( ${#thisStringHex_array[@]} + ${padding})); i++)); do
		charHex_cipher=${cipherHex_array[$(( $i % 11 ))]}

		charHex=${thisStringHex_array[$i]}
		printf "%02X" "$(( 0x${charHex_cipher} ^ 0x${charHex:-00} ))" | xxd -r -p > /dev/stdout
	done
)

# If account exists, do not create it.
if ( dscl . -list /Users | grep -q "$EBSAdminUser" ); then
echo "$EBSAdminUser account already exists."
else
# Creates and elevates account to admin. 
highestUID=$(dscl . -list /Users UniqueID | sort -nr -k 2 | head -1 | grep -oE '[0-9]+$')
((newUserUID=highestUID+1))
echo "Creating $EBSAdminUser account (UID $newUserUID)."
createUserAccount "$EBSAdminUser" "$EBSAdminPassword" $newUserUID
sudo /usr/bin/dscl . -append /Groups/admin GroupMembership "$EBSAdminUser"
fi

# Sets password for user account using variables above.
echo "Setting password for $EBSAdminUser account."
sudo /usr/bin/dscl . -passwd "/Users/$EBSAdminUser" "$EBSAdminPassword"

# Invisible directory for staging.
stagingDir="/Users/Shared/._ec2-auto-login"
sudo mkdir -p "$stagingDir"
sudo chown 501:20 "$stagingDir"
sudo chmod -R 775 "$stagingDir"


# Enable auto-login for user.
sudo defaults write "/Library/Preferences/com.apple.loginwindow" autoLoginUser "$EBSAdminUser"

# Set name of Secret for enroll-ec2-mac script to run.
sudo defaults write "/Users/$EBSAdminUser/Library/Preferences/com.amazon.dsx.ec2.enrollment.automation" MMSecret "$credentialID"
sudo chown $EBSAdminUser:staff "/Users/$EBSAdminUser/Library/Preferences/com.amazon.dsx.ec2.enrollment.automation.plist"

kcpasswordEncode "$EBSAdminPassword" > "$stagingDir/kcpassword"
sudo cp "$stagingDir/kcpassword" "/etc/"
sudo chown root:wheel "/etc/kcpassword"
sudo chmod u=rw,go= "/etc/kcpassword"

# Downloads and installs the latest enroll-ec2-mac.scpt, including loading the LaunchAgent for setup.
sudo curl -H 'Accept: application/vnd.github.v3.raw' https://api.github.com/repos/aws-samples/amazon-ec2-mac-mdm-enrollment-automation/contents/enroll-ec2-mac.scpt | sudo tee "/Users/Shared/enroll-ec2-mac.scpt" > /dev/null ; sudo chmod +x "/Users/Shared/enroll-ec2-mac.scpt"; osascript "/Users/Shared/enroll-ec2-mac.scpt" --setup && echo "Setup Done" || echo "Setup Not Done"

# Enable Screen Sharing for remote GUI connection.
sudo launchctl enable system/com.apple.screensharing ; sudo launchctl load -w /System/Library/LaunchDaemons/com.apple.screensharing.plist

# Sets a flag to prevent script from attempting to run twice.
if [ -f "$stagingDir/.userSetupComplete" ]; then
# Clean up auto-login database.
rm -rf "$stagingDir/kcpassword"
sleep 1
else
touch "$stagingDir/.userSetupComplete"
sudo launchctl reboot
fi

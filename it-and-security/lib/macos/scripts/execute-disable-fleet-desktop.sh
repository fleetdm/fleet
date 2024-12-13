#!/bin/sh


# execute.disable.fleet.desktop @2024 Fleet Device Management
# Brock Walters (brock@fleetdm.com)


# variables
dskplst='/Library/LaunchDaemons/com.fleet.disable.desktop.plist'
dskscpt='/private/tmp/disable.fleet.desktop.sh'
fltplst='/Library/LaunchDaemons/com.fleetdm.orbit.plist'


# check Fleet Desktop, exit if not enabled
if /usr/libexec/PlistBuddy -c 'print EnvironmentVariables:ORBIT_FLEET_DESKTOP' "$fltplst" | /usr/bin/grep -iq 'false'
then
    printf "Fleet Desktop is not enabled on this host. Exiting...\n"; exit
else
    printf "Disabling Fleet Desktop...\n"
fi


printf "Writing out disable Fleet Desktop script...\n"
/bin/cat << 'EOF' > "$dskscpt"
#!/bin/sh

# logging
cpuname="$(/usr/sbin/scutil --get ComputerName)"
srlnmbr="$(/usr/libexec/PlistBuddy -c 'print 0:serial-number' /dev/stdin <<< "$(/usr/sbin/ioreg -ar -d 1 -k 'IOPlatformSerialNumber')")"
usrcrnt="$(/usr/bin/stat -f %Su /dev/console)"
logexec="$(/usr/bin/basename "$0")"
logpath="/private/var/log/${logexec%.*}.log"
logpipe="/private/tmp/${logexec%.*}.pipe"

/usr/bin/mkfifo "$logpipe"
/usr/bin/tee -a < "$logpipe" "$logpath" &
exec &> "$logpipe"
printf "$(/bin/date "+%Y-%m-%dT%H:%M:%S") [START] logging %s\n   computer name: %s\n   serial number: %s\n   current user: %s\n" "$logexec" "$cpuname" "$srlnmbr" "$usrcrnt"  >> "$logpath"

logalrt(){
>&2 printf "$(/bin/date "+%Y-%m-%dT%H:%M:%S") [ALERT] %s" >> "$logpath"
}

logexit(){
>&2 printf "$(/bin/date "+%Y-%m-%dT%H:%M:%S") [STOP] logging %s" "$logexec" >> "$logpath"
/bin/rm -f "$logpipe"; /usr/bin/pkill -ail tee > /dev/null
}

loginfo(){
>&2 printf "$(/bin/date "+%Y-%m-%dT%H:%M:%S") [INFO] %s" >> "$logpath"
}

# variables
count=0
dskplst='/Library/LaunchDaemons/com.fleet.disable.desktop.plist'
dskscpt='/private/tmp/disable.fleet.desktop.sh'
fltplst='/Library/Launchdaemons/com.fleetdm.orbit.plist'

# operations
/usr/libexec/PlistBuddy -c 'set EnvironmentVariables:ORBIT_FLEET_DESKTOP false' "$fltplst"; /bin/sleep 10
/bin/launchctl bootout system "$fltplst"; /bin/sleep 3; /bin/launchctl bootstrap system "$fltplst"; /bin/sleep 3
logalrt; printf "Fleet Desktop disabled.\n"

while true
do
    if /bin/launchctl list | /usr/bin/grep -iq 'com.fleetdm.orbit' && /usr/libexec/PlistBuddy -c 'print EnvironmentVariables:ORBIT_FLEET_DESKTOP' "$fltplst" | /usr/bin/grep -iq 'false'
    then
        loginfo; printf "fleetd restarted.\n"
        loginfo; printf "Attempting to bootout com.fleet.disable.desktop...\n"
        loginfo; printf "Removing:\n   %s\n   %s\n" "$dskplst" "$dskscpt"
        logexit; /bin/rm -f "$dskplst" "$dskscpt" &
        /bin/launchctl bootout system/com.fleet.disable.desktop
    else
        count=$((count+1))

        if [ "$count" -gt 60 ]
        then
            logalrt; printf "Unable to restart fleetd. Exiting...\n"; logexit; exit 
        else
            loginfo; printf "Waiting for fleetd...\n"; /bin/sleep 1; continue
        fi
    fi
done
EOF
/bin/chmod 755 "$dskscpt"; /usr/sbin/chown 0:0 "$dskscpt"


printf "Writing out disable Fleet Desktop Launch Daemon...\n"
/bin/cat << 'EOF' > "$dskplst"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>Label</key>
            <string>com.fleet.disable.desktop</string>
        <key>ProgramArguments</key>
        <array>
            <string>/bin/sh</string>
            <string>/private/tmp/disable.fleet.desktop.sh</string>
        </array>
        <key>RunAtLoad</key>
            <true/>
        <key>AbandonProcessGroup</key>
            <true/>
        <key>StandardErrorPath</key>
            <string>/dev/null</string>
        <key>StandardOutPath</key>
            <string>/dev/null</string>
    </dict>
</plist>
EOF
/bin/chmod 644 "$dskplst"; /usr/sbin/chown 0:0 "$dskplst"


printf "Waiting for child process to disable Fleet Desktop...\n"; /bin/sleep 10
if /bin/launchctl bootstrap system "$dskplst" | /usr/bin/grep 'Bootstrap failed'
then
    printf "... child process failed. Exiting...\n"; exit
else
    printf "... Ok.\n"; exit
fi


# re-enable
# sudo /bin/launchctl bootout system /Library/LaunchDaemons/com.fleet.disable.desktop.plist; /bin/sleep 3
# sudo /usr/libexec/PlistBuddy -c 'set EnvironmentVariables:ORBIT_FLEET_DESKTOP true' /Library/Launchdaemons/com.fleetdm.orbit.plist
# sudo /bin/launchctl bootout system /Library/LaunchDaemons/com.fleetdm.orbit.plist
# sudo /bin/launchctl bootstrap system /Library/LaunchDaemons/com.fleetdm.orbit.plist
# sudo rm -rf /private/tmp/execute.disable.fleet.desktop.pipe /private/tmp/disable.fleet.desktop.sh /Library/LaunchDaemons/com.fleet.disable.desktop.plist /private/var/log/execute.disable.fleet.desktop.log





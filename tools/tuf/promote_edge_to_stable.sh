#!/bin/bash


component=$1
version=$2

if [[ -z $component || -z $version ]]; then
    echo "Usage: $0 <component> <version>"
    exit 1
fi

if [[ ! -d "./repository" ]]; then
    echo "Directory ./repository doesn't exist"
    exit 1
fi

version_parts=(${version//./ })
major=${version_parts[0]}  
minor=${version_parts[1]}  

echo "Promoting $component from edge to stable, version='$version'"
echo "Press any key to continue..."
read -s -n 1

case $1 in
    orbit)
        fleetctl updates add --target ./repository/targets/orbit/macos/edge/orbit --platform macos --name orbit --version $version -t $major.$minor -t $major -t stable
        fleetctl updates add --target ./repository/targets/orbit/linux/edge/orbit --platform linux --name orbit --version $version -t $major.$minor -t $major -t stable
        fleetctl updates add --target ./repository/targets/orbit/windows/edge/orbit.exe --platform windows --name orbit --version $version -t $major.$minor -t $major -t stable
        ;;
    desktop)
        fleetctl updates add --target ./repository/targets/desktop/macos/edge/desktop.app.tar.gz --platform macos --name desktop --version $version -t $major.$minor -t $major -t stable
        fleetctl updates add --target ./repository/targets/desktop/linux/edge/desktop.tar.gz --platform linux --name desktop --version $version -t $major.$minor -t $major -t stable
        fleetctl updates add --target ./repository/targets/desktop/windows/edge/fleet-desktop.exe --platform windows --name desktop --version $version -t $major.$minor -t $major -t stable
        ;;
    osqueryd)
        fleetctl updates add --target ./repository/targets/osqueryd/macos-app/edge/osqueryd.app.tar.gz --platform macos-app --name osqueryd --version $version -t $major.$minor -t $major -t stable
        fleetctl updates add --target ./repository/targets/osqueryd/linux/edge/osqueryd --platform linux --name osqueryd --version $version -t $major.$minor -t $major -t stable
        fleetctl updates add --target ./repository/targets/osqueryd/windows/edge/osqueryd.exe --platform windows --name osqueryd --version $version -t $major.$minor -t $major -t stable
        ;;
    nudge)
        fleetctl updates add --target ./repository/targets/nudge/macos/edge/nudge.app.tar.gz --platform macos --name nudge --version $version -t stable
        ;;
    swiftDialog)
        fleetctl updates add --target ./repository/targets/swiftDialog/macos/edge/swiftDialog.app.tar.gz --platform macos --name swiftDialog --version $version -t stable
        ;;
    *)
        echo Unknown component $1
        exit 1
        ;;
esac

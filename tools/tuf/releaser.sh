#!/bin/bash

#
# For usage documentation, see the README.md.
#

set -e

#
# Input environment variables:
#
# AWS_PROFILE
# TUF_DIRECTORY
# COMPONENT
# ACTION
# VERSION
# KEYS_SOURCE_DIRECTORY
# TARGETS_PASSPHRASE_1PASSWORD_PATH
# SNAPSHOT_PASSPHRASE_1PASSWORD_PATH
# TIMESTAMP_PASSPHRASE_1PASSWORD_PATH
# GITHUB_USERNAME
# GITHUB_TOKEN_1PASSWORD_PATH
# SKIP_PR_AND_TAG_PUSH
#

#
# Dev environment variables:
# PUSH_TO_REMOTE
# GIT_REPOSITORY_DIRECTORY
#

clean_up () {
    echo "Cleaning up directories..."

    # Make sure (best effort) to remove the keys after we are done.
    rm -rf "$KEYS_DIRECTORY"
    rm -rf "$ARTIFACTS_DOWNLOAD_DIRECTORY"
    rm -rf "$GO_TOOLS_DIRECTORY"
    ARG=$?
    exit $ARG
} 

setup () {
    echo "Running setup..."

    GO_TOOLS_DIRECTORY=$(mktemp -d)
    ARTIFACTS_DOWNLOAD_DIRECTORY=$(mktemp -d)
    SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
    REPOSITORY_DIRECTORY=$TUF_DIRECTORY/repository
    STAGED_DIRECTORY=$TUF_DIRECTORY/staged
    KEYS_DIRECTORY=$TUF_DIRECTORY/keys
    if [[ -z $GIT_REPOSITORY_DIRECTORY ]]; then
        GIT_REPOSITORY_DIRECTORY=$( realpath "$SCRIPT_DIR/../.." )
    fi

    mkdir -p "$REPOSITORY_DIRECTORY"
    mkdir -p "$STAGED_DIRECTORY"
    cp -r "$KEYS_SOURCE_DIRECTORY" "$KEYS_DIRECTORY"

    if ! aws sts get-caller-identity &> /dev/null; then
        aws sso login
        prompt "AWS SSO login was successful, press any key to continue..."
    fi

    # GITHUB_TOKEN is only necessary when releasing to edge.
    if [[ -n $GITHUB_TOKEN_1PASSWORD_PATH ]]; then
        GITHUB_TOKEN=$(op read "op://$GITHUB_TOKEN_1PASSWORD_PATH")
    fi

    # These need to be exported for use by `fleetctl updates` commands.
    FLEET_TARGETS_PASSPHRASE=$(op read "op://$TARGETS_PASSPHRASE_1PASSWORD_PATH")
    export FLEET_TARGETS_PASSPHRASE
    FLEET_SNAPSHOT_PASSPHRASE=$(op read "op://$SNAPSHOT_PASSPHRASE_1PASSWORD_PATH")
    export FLEET_SNAPSHOT_PASSPHRASE
    FLEET_TIMESTAMP_PASSPHRASE=$(op read "op://$TIMESTAMP_PASSPHRASE_1PASSWORD_PATH")
    export FLEET_TIMESTAMP_PASSPHRASE

    go build -o "$GO_TOOLS_DIRECTORY/replace" "$SCRIPT_DIR/../../tools/tuf/replace" 
    go build -o "$GO_TOOLS_DIRECTORY/download-artifacts" "$SCRIPT_DIR/../../tools/tuf/download-artifacts"
}

pull_from_remote () {
    echo "Pulling repository from tuf.fleetctl.com... (--dryrun first)"
    aws s3 sync s3://fleet-tuf-repo "$REPOSITORY_DIRECTORY" --exact-timestamps --dryrun
    prompt "If the --dryrun looks good, press any key to continue... (no output means nothing to update)"
    aws s3 sync s3://fleet-tuf-repo "$REPOSITORY_DIRECTORY" --exact-timestamps
}

promote_component_edge_to_stable () {
    component_name=$1
    component_version=$2

    IFS='.' read -r -a version_parts <<< "$component_version"
    major=${version_parts[0]}  
    minor=${version_parts[1]}  

    pushd "$TUF_DIRECTORY"
    case $component_name in
        orbit)
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/orbit/macos/edge/orbit" --platform macos --name orbit --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/orbit/linux/edge/orbit" --platform linux --name orbit --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/orbit/windows/edge/orbit.exe" --platform windows --name orbit --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            ;;
        desktop)
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/macos/edge/desktop.app.tar.gz" --platform macos --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/linux/edge/desktop.tar.gz" --platform linux --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/windows/edge/fleet-desktop.exe" --platform windows --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            ;;
        osqueryd)
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/macos-app/edge/osqueryd.app.tar.gz" --platform macos-app --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/linux/edge/osqueryd" --platform linux --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/windows/edge/osqueryd.exe" --platform windows --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            ;;
        *)
            echo "Unknown component $component_name"
            exit 1
            ;;
    esac
    popd
}

promote_edge_to_stable () {
    cd "$REPOSITORY_DIRECTORY"
    if [[ $COMPONENT == "fleetd" ]]; then
        echo "Promoting fleetd from edge to stable..."
        promote_component_edge_to_stable orbit "$VERSION"
        promote_component_edge_to_stable desktop "$VERSION"
    elif [[ $COMPONENT == "osqueryd" ]]; then
        echo "Promoting osqueryd from edge to stable..."
        promote_component_edge_to_stable osqueryd "$VERSION"
    else
        echo "Unsupported component: $COMPONENT"
        exit 1
    fi
}

release_fleetd_to_edge () {
    echo "Releasing fleetd to edge..."
    BRANCH_NAME="release-fleetd-v$VERSION"
    ORBIT_TAG="orbit-v$VERSION"
    if [[ "$SKIP_PR_AND_TAG_PUSH" != "1" ]]; then
        prompt "A PR for bumping the fleetd version will be created to trigger a Github Action that will build 'Fleet Desktop'. Press any key to continue..."
        pushd "$GIT_REPOSITORY_DIRECTORY"
        git checkout -b "$BRANCH_NAME"
        make changelog-orbit version="$VERSION"
        ORBIT_CHANGELOG=orbit/CHANGELOG.md
        "$GO_TOOLS_DIRECTORY/replace" .github/workflows/generate-desktop-targets.yml "FLEET_DESKTOP_VERSION: .+\n" "FLEET_DESKTOP_VERSION: $VERSION\n"
        git add .github/workflows/generate-desktop-targets.yml "$ORBIT_CHANGELOG"
        git commit -m "Release fleetd $VERSION"
        git push origin "$BRANCH_NAME"
        open "https://github.com/fleetdm/fleet/pull/new/$BRANCH_NAME"
        prompt "Press any key to continue after the PR is created..."
        prompt "A 'git tag' will be created to trigger a Github Action to build orbit, press any key to continue..."
        git tag "$ORBIT_TAG"
        git push origin "$ORBIT_TAG"
        popd
    fi
    DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY="$ARTIFACTS_DOWNLOAD_DIRECTORY/desktop"
    mkdir -p "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY"
    "$GO_TOOLS_DIRECTORY/download-artifacts" desktop \
        --git-branch "$BRANCH_NAME" \
        --output-directory "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY" \
        --github-username "$GITHUB_USERNAME" --github-api-token "$GITHUB_TOKEN" \
        --retry
    ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY="$ARTIFACTS_DOWNLOAD_DIRECTORY/orbit"
    mkdir -p "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY"
    "$GO_TOOLS_DIRECTORY/download-artifacts" orbit \
        --git-tag "$ORBIT_TAG" \
        --output-directory "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY" \
        --github-username "$GITHUB_USERNAME" --github-api-token "$GITHUB_TOKEN" \
        --retry
    pushd "$TUF_DIRECTORY"
    fleetctl updates add --target "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY/macos/orbit" --platform macos --name orbit --version "$VERSION" -t edge
    fleetctl updates add --target "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY/linux/orbit" --platform linux --name orbit --version "$VERSION" -t edge
    fleetctl updates add --target "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY/windows/orbit.exe" --platform windows --name orbit --version "$VERSION" -t edge
    fleetctl updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/macos/desktop.app.tar.gz" --platform macos --name desktop --version "$VERSION" -t edge
    fleetctl updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/linux/desktop.tar.gz" --platform linux --name desktop --version "$VERSION" -t edge
    fleetctl updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/windows/fleet-desktop.exe" --platform windows --name desktop --version "$VERSION" -t edge
    popd
}

release_osqueryd_to_edge () {
    echo "Releasing osqueryd to edge..."
    prompt "A branch and PR for bumping the osquery version will be created. Press any key to continue..."
    BRANCH_NAME=release-osqueryd-v$VERSION
    if [[ "$SKIP_PR_AND_TAG_PUSH" != "1" ]]; then
        pushd "$GIT_REPOSITORY_DIRECTORY"
        git checkout -b "$BRANCH_NAME"
        "$GO_TOOLS_DIRECTORY/replace" .github/workflows/generate-osqueryd-targets.yml "OSQUERY_VERSION: .+\n" "OSQUERY_VERSION: $VERSION\n"
        git add .github/workflows/generate-osqueryd-targets.yml
        git commit -m "Bump osqueryd version to $VERSION"
        git push origin "$BRANCH_NAME"
        open "https://github.com/fleetdm/fleet/pull/new/$BRANCH_NAME"
        prompt "Press any key to continue after the PR is created..."
        popd
    fi
    OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY="$ARTIFACTS_DOWNLOAD_DIRECTORY/osqueryd"
    mkdir -p "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY"
    "$GO_TOOLS_DIRECTORY/download-artifacts" osqueryd \
        --git-branch "$BRANCH_NAME" \
        --output-directory "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY" \
        --github-username "$GITHUB_USERNAME" \
        --github-api-token "$GITHUB_TOKEN" \
        --retry
    pushd "$TUF_DIRECTORY"
    fleetctl updates add --target "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY/macos/osqueryd.app.tar.gz" --platform macos-app --name osqueryd --version "$VERSION" -t edge
    fleetctl updates add --target "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY/linux/osqueryd" --platform linux --name osqueryd --version "$VERSION" -t edge
    fleetctl updates add --target "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY/windows/osqueryd.exe" --platform windows --name osqueryd --version "$VERSION" -t edge
    popd
}

release_to_edge () {
    if [[ $COMPONENT == "fleetd" ]]; then
        release_fleetd_to_edge
    elif [[ $COMPONENT == "osqueryd" ]]; then
        release_osqueryd_to_edge
    else
        echo "Unsupported component: $COMPONENT"
        exit 1
    fi
}

push_to_remote () {
    echo "Running --dryrun push of repository to tuf.fleetctl.com..."
    aws s3 sync "$REPOSITORY_DIRECTORY" s3://fleet-tuf-repo --dryrun
    if [[ $PUSH_TO_REMOTE == "1" ]]; then
        echo "WARNING: This step will push the release to tuf.fleetctl.com (production)..."
        prompt "If the --dryrun looks good, press any key to continue..."
        aws s3 sync "$REPOSITORY_DIRECTORY" s3://fleet-tuf-repo
        echo "Release has been pushed!"
        echo "NOTE: You might see some clients failing to upgrade due to some sha256 mismatches."
        echo "These temporary failures are expected because it takes some time for caches to be invalidated (these errors should go away after ~15-30 minutes)."
    else
        echo "PUSH_TO_REMOTE not set to 1, so not pushing."
    fi
}

prompt () {
    printf "%s\n" "$1"
    read -r -s -n 1
}

setup_to_become_publisher () {
    echo "Running setup to become publisher..."

    REPOSITORY_DIRECTORY=$TUF_DIRECTORY/repository
    STAGED_DIRECTORY=$TUF_DIRECTORY/staged
    KEYS_DIRECTORY=$TUF_DIRECTORY/keys
    mkdir -p "$REPOSITORY_DIRECTORY"
    mkdir -p "$STAGED_DIRECTORY"
    mkdir -p "$KEYS_DIRECTORY"
    if ! aws sts get-caller-identity &> /dev/null; then
        aws sso login
        prompt "AWS SSO login was successful, press any key to continue..."
    fi
    # These need to be exported for use by `tuf` commands.
    FLEET_TARGETS_PASSPHRASE=$(op read "op://$TARGETS_PASSPHRASE_1PASSWORD_PATH")
    export TUF_TARGETS_PASSPHRASE=$FLEET_TARGETS_PASSPHRASE
    FLEET_SNAPSHOT_PASSPHRASE=$(op read "op://$SNAPSHOT_PASSPHRASE_1PASSWORD_PATH")
    export TUF_SNAPSHOT_PASSPHRASE=$FLEET_SNAPSHOT_PASSPHRASE
    FLEET_TIMESTAMP_PASSPHRASE=$(op read "op://$TIMESTAMP_PASSPHRASE_1PASSWORD_PATH")
    export TUF_TIMESTAMP_PASSPHRASE=$FLEET_TIMESTAMP_PASSPHRASE
}

if [[ $ACTION == "generate-signing-keys" ]]; then
    setup_to_become_publisher
    pull_from_remote
    cd "$TUF_DIRECTORY"
    tuf gen-key targets && echo
    tuf gen-key snapshot && echo
    tuf gen-key timestamp && echo
    echo "Keys have been generated, now do the following actions:"
    echo "- Share '$TUF_DIRECTORY/staged/root.json' with Fleet member with the 'root' role, who will sign with its root key and push it to the remote repository."
    echo "- Store the '$TUF_DIRECTORY/keys' folder (that contains the encrypted keys) on a USB flash drive that you will ONLY use for releasing fleetd updates."
    exit 0
fi

print_reminder () {
    if [[ $ACTION == "release-to-edge" ]]; then
        if [[ $COMPONENT == "fleetd" ]]; then
            prompt "Make sure to install fleetd with '--orbit-channel=edge --desktop-channel=edge' on a Linux, Windows and macOS VM. (To smoke test the release.) Press any key to continue..."
        elif [[ $COMPONENT == "osqueryd" ]]; then
            prompt "Make sure to install fleetd with '--osqueryd-channel=edge' on a Linux, Windows and macOS VM. (To smoke test the release.) Press any key to continue..."
        fi
    elif [[ $ACTION == "promote-edge-to-stable" ]]; then
        if [[ $COMPONENT == "fleetd" ]]; then
            prompt "Make sure to install fleetd with '--orbit-channel=stable --desktop-channel=stable' on a Linux, Windows and macOS VM. (To smoke test the release.) Press any key to continue..."
        elif [[ $COMPONENT == "osqueryd" ]]; then
            prompt "Make sure to install fleetd with '--osqueryd-channel=stable' on a Linux, Windows and macOS VM. (To smoke test the release.) Press any key to continue..."
        fi
    else
        echo "Unsupported action: $ACTION"
    fi
}

trap clean_up EXIT
print_reminder
setup
pull_from_remote

if [[ $ACTION == "release-to-edge" ]]; then
    release_to_edge
elif [[ $ACTION == "promote-edge-to-stable" ]]; then
    promote_edge_to_stable
else
    echo "Unsupported action: $ACTION"
    exit 1
fi

push_to_remote
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
        prompt "You need to login to AWS using the cli."
        aws sso login
        prompt "AWS SSO login was successful."
    fi

    # GITHUB_TOKEN is only necessary when releasing to edge.
    if [[ -n $GITHUB_TOKEN_1PASSWORD_PATH ]]; then
        GITHUB_TOKEN=$(op read "op://$GITHUB_TOKEN_1PASSWORD_PATH")
    fi

    # We only need to be logged in to github when releasing to edge.
    if [[ $ACTION == "release-to-edge" ]]; then
        if ! gh auth status >/dev/null 2>&1; then
            prompt "You need to login to Github using the cli."
            gh auth login
            prompt "Github login was successful."
        fi
    fi

    #
    # Passphrases need to be exported for use by `fleetctl updates` commands.
    #

    if [[ $ACTION == "release-to-edge" ]] || [[ $ACTION == "promote-edge-to-stable"  ]]; then
        FLEET_TARGETS_PASSPHRASE=$(op read "op://$TARGETS_PASSPHRASE_1PASSWORD_PATH")
        export FLEET_TARGETS_PASSPHRASE
        FLEET_SNAPSHOT_PASSPHRASE=$(op read "op://$SNAPSHOT_PASSPHRASE_1PASSWORD_PATH")
        export FLEET_SNAPSHOT_PASSPHRASE
        FLEET_TIMESTAMP_PASSPHRASE=$(op read "op://$TIMESTAMP_PASSPHRASE_1PASSWORD_PATH")
        export FLEET_TIMESTAMP_PASSPHRASE
    elif [[ $ACTION == "update-timestamp" ]]; then
        FLEET_TIMESTAMP_PASSPHRASE=$(op read "op://$TIMESTAMP_PASSPHRASE_1PASSWORD_PATH")
        export FLEET_TIMESTAMP_PASSPHRASE
    fi

    go build -o "$GO_TOOLS_DIRECTORY/replace" "$SCRIPT_DIR/../../tools/tuf/replace"
    go build -o "$GO_TOOLS_DIRECTORY/download-artifacts" "$SCRIPT_DIR/../../tools/tuf/download-artifacts"
}

pull_from_remote () {
    echo "Pulling repository from tuf.fleetctl.com... (--dryrun first)"
    aws s3 sync s3://fleet-tuf-repo "$REPOSITORY_DIRECTORY" --exact-timestamps --dryrun
    prompt "Check if the above --dry-run looks good (no output means nothing to update)."
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
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/orbit/linux-arm64/edge/orbit" --platform linux-arm64 --name orbit --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/orbit/windows/edge/orbit.exe" --platform windows --name orbit --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            ;;
        desktop)
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/macos/edge/desktop.app.tar.gz" --platform macos --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/linux/edge/desktop.tar.gz" --platform linux --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/linux-arm64/edge/desktop.tar.gz" --platform linux-arm64 --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/windows/edge/fleet-desktop.exe" --platform windows --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            ;;
        osqueryd)
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/macos-app/edge/osqueryd.app.tar.gz" --platform macos-app --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/linux/edge/osqueryd" --platform linux --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            fleetctl updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/linux-arm64/edge/osqueryd" --platform linux-arm64 --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
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
        prompt "A PR will be created to trigger a Github Action to build desktop."
        pushd "$GIT_REPOSITORY_DIRECTORY"
        git checkout -b "$BRANCH_NAME"
        make changelog-orbit version="$VERSION"
        ORBIT_CHANGELOG=orbit/CHANGELOG.md
        "$GO_TOOLS_DIRECTORY/replace" .github/workflows/generate-desktop-targets.yml "FLEET_DESKTOP_VERSION: .+\n" "FLEET_DESKTOP_VERSION: $VERSION\n"
        git add .github/workflows/generate-desktop-targets.yml "$ORBIT_CHANGELOG"
        git commit -m "Release fleetd $VERSION"
        git push origin "$BRANCH_NAME"
        gh pr create -f -B main -t "Release fleetd $VERSION"
        prompt "A 'git tag' will be created to trigger a Github Action to build orbit."
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
    fleetctl updates add --target "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY/linux-arm64/orbit" --platform linux-arm64 --name orbit --version "$VERSION" -t edge
    fleetctl updates add --target "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY/windows/orbit.exe" --platform windows --name orbit --version "$VERSION" -t edge
    fleetctl updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/macos/desktop.app.tar.gz" --platform macos --name desktop --version "$VERSION" -t edge
    fleetctl updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/linux/desktop.tar.gz" --platform linux --name desktop --version "$VERSION" -t edge
    fleetctl updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/linux-arm64/desktop.tar.gz" --platform linux-arm64 --name desktop --version "$VERSION" -t edge
    fleetctl updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/windows/fleet-desktop.exe" --platform windows --name desktop --version "$VERSION" -t edge
    popd
}

release_osqueryd_to_edge () {
    echo "Releasing osqueryd to edge..."
    prompt "A branch and PR for bumping the osquery version will be created."
    BRANCH_NAME=release-osqueryd-v$VERSION
    if [[ "$SKIP_PR_AND_TAG_PUSH" != "1" ]]; then
        pushd "$GIT_REPOSITORY_DIRECTORY"
        git checkout -b "$BRANCH_NAME"
        "$GO_TOOLS_DIRECTORY/replace" .github/workflows/generate-osqueryd-targets.yml "OSQUERY_VERSION: .+\n" "OSQUERY_VERSION: $VERSION\n"
        git add .github/workflows/generate-osqueryd-targets.yml
        git commit -m "Bump osqueryd version to $VERSION"
        git push origin "$BRANCH_NAME"
        prompt "A PR will be created to trigger a Github Action to build osqueryd."
        gh pr create -f -B main -t "Release osqueryd $VERSION"
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
    fleetctl updates add --target "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY/linux-arm64/osqueryd" --platform linux-arm64 --name osqueryd --version "$VERSION" -t edge
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

update_timestamp () {
    pushd "$TUF_DIRECTORY"
    fleetctl updates timestamp
    popd
}

push_to_remote () {
    echo "Running --dryrun push of repository to tuf.fleetctl.com..."
    aws s3 sync "$REPOSITORY_DIRECTORY" s3://fleet-tuf-repo --dryrun
    if [[ $PUSH_TO_REMOTE == "1" ]]; then
        echo "WARNING: This step will push the release to tuf.fleetctl.com (production)..."
        prompt "Check if the above --dry-run looks good."
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
    printf "Type 'yes' to continue... "
    while read -r word;
    do
        if [[ "$word" == "yes" ]]; then
            printf "\n"
            return
        fi
    done
}

print_reminder () {
    if [[ $ACTION == "release-to-edge" ]]; then
        if [[ $COMPONENT == "fleetd" ]]; then
            prompt "Make sure to install fleetd with '--orbit-channel=edge --desktop-channel=edge' on a Linux, Windows and macOS VM. (To smoke test the release.)"
        elif [[ $COMPONENT == "osqueryd" ]]; then
            prompt "Make sure to install fleetd with '--osqueryd-channel=edge' on a Linux, Windows and macOS VM. (To smoke test the release.)"
        fi
    elif [[ $ACTION == "promote-edge-to-stable" ]]; then
        if [[ $COMPONENT == "fleetd" ]]; then
            prompt "Make sure to install fleetd with '--orbit-channel=stable --desktop-channel=stable' on a Linux, Windows and macOS VM. (To smoke test the release.)"
        elif [[ $COMPONENT == "osqueryd" ]]; then
            prompt "Make sure to install fleetd with '--osqueryd-channel=stable' on a Linux, Windows and macOS VM. (To smoke test the release.)"
        fi
    elif [[ $ACTION == "update-timestamp" ]]; then
        :
    elif [[ $ACTION != "update-timestamp" ]]; then
        echo "Unsupported action: $ACTION"
        exit 1
    fi
}

fleetctl_version_check () {
    which fleetctl
    fleetctl --version
    prompt "Make sure the fleetctl executable and version are correct."
}

trap clean_up EXIT
print_reminder
fleetctl_version_check
setup

pull_from_remote

if [[ $ACTION == "release-to-edge" ]]; then
    release_to_edge
elif [[ $ACTION == "promote-edge-to-stable" ]]; then
    promote_edge_to_stable
elif [[ $ACTION == "update-timestamp" ]]; then
    update_timestamp
else
    echo "Unsupported action: $ACTION"
    exit 1
fi

push_to_remote
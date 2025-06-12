#!/bin/bash

#
# For usage documentation, see the README.md.
#

set -e

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

pull_from_staging () {
    echo "Pulling repository from updates-staging.fleetctl.com... (--dryrun first)"
    rclone sync --verbose --checksum r2://updates-staging "$REPOSITORY_DIRECTORY" --dry-run
    prompt "Check if the above --dry-run looks good (no output means nothing to update)."
    rclone sync --verbose --checksum r2://updates-staging "$REPOSITORY_DIRECTORY"
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
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/orbit/macos/edge/orbit" --platform macos --name orbit --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/orbit/linux/edge/orbit" --platform linux --name orbit --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/orbit/linux-arm64/edge/orbit" --platform linux-arm64 --name orbit --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/orbit/windows/edge/orbit.exe" --platform windows --name orbit --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/orbit/windows-arm64/edge/orbit.exe" --platform windows-arm64 --name orbit --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            ;;
        desktop)
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/macos/edge/desktop.app.tar.gz" --platform macos --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/linux/edge/desktop.tar.gz" --platform linux --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/linux-arm64/edge/desktop.tar.gz" --platform linux-arm64 --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/windows/edge/fleet-desktop.exe" --platform windows --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/desktop/windows-arm64/edge/fleet-desktop.exe" --platform windows-arm64 --name desktop --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            ;;
        osqueryd)
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/macos-app/edge/osqueryd.app.tar.gz" --platform macos-app --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/linux/edge/osqueryd" --platform linux --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/linux-arm64/edge/osqueryd" --platform linux-arm64 --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/windows/edge/osqueryd.exe" --platform windows --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
            "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$REPOSITORY_DIRECTORY/targets/osqueryd/windows-arm64/edge/osqueryd.exe" --platform windows-arm64 --name osqueryd --version "$component_version" -t "$major.$minor" -t "$major" -t stable
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
    ORBIT_TAG="orbit-v$VERSION"
    prompt "A tag will be pushed to trigger a Github Action to build desktop and orbit."
    pushd "$GIT_REPOSITORY_DIRECTORY"
    git tag "$ORBIT_TAG"
    git push origin "$ORBIT_TAG"
    if [[ "$SKIP_PR" != "1" ]]; then
        create_fleetd_release_pr
    fi
    popd
    DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY="$ARTIFACTS_DOWNLOAD_DIRECTORY/desktop"
    mkdir -p "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY"
    "$GO_TOOLS_DIRECTORY/download-artifacts" desktop \
        --git-tag "$ORBIT_TAG" \
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
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY/macos/orbit" --platform macos --name orbit --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY/linux/orbit" --platform linux --name orbit --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY/linux-arm64/orbit" --platform linux-arm64 --name orbit --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY/windows/orbit.exe" --platform windows --name orbit --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$ORBIT_ARTIFACT_DOWNLOAD_DIRECTORY/windows-arm64/orbit.exe" --platform windows-arm64 --name orbit --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/macos/desktop.app.tar.gz" --platform macos --name desktop --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/linux/desktop.tar.gz" --platform linux --name desktop --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/linux-arm64/desktop.tar.gz" --platform linux-arm64 --name desktop --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/windows/fleet-desktop.exe" --platform windows --name desktop --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$DESKTOP_ARTIFACT_DOWNLOAD_DIRECTORY/windows-arm64/fleet-desktop.exe" --platform windows-arm64 --name desktop --version "$VERSION" -t edge
    popd
}

create_fleetd_release_pr () {
    echo "Creating a PR against main for fleetd release changelog..."
    BRANCH_NAME=release-fleetd-v$VERSION
    pushd "$GIT_REPOSITORY_DIRECTORY"
    # Create a branch to make the changelog update on.
    git checkout -b "${BRANCH_NAME}-changelog"
    make changelog-orbit version="$VERSION"
    ORBIT_CHANGELOG=orbit/CHANGELOG.md
    git add "$ORBIT_CHANGELOG"
    git commit -m "Release fleetd $VERSION"
    # Checkout the main branch.
    git checkout main
    # Create a new branch to cherry pick the changelog commit to.
    git checkout -b "$BRANCH_NAME"
    # Cherry pick the changelog commit to the new branch.
    git cherry-pick "${BRANCH_NAME}-changelog"
    # Create a new PR with the changelog.
    gh pr create -f -B main -t "Update changelog for fleetd $VERSION release"
    # Delete the changelog branch.
    git branch -D "${BRANCH_NAME}-changelog"
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
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY/macos/osqueryd.app.tar.gz" --platform macos-app --name osqueryd --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY/linux/osqueryd" --platform linux --name osqueryd --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY/linux-arm64/osqueryd" --platform linux-arm64 --name osqueryd --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY/windows/osqueryd.exe" --platform windows --name osqueryd --version "$VERSION" -t edge
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates add --target "$OSQUERYD_ARTIFACT_DOWNLOAD_DIRECTORY/windows-arm64/osqueryd.exe" --platform windows-arm64 --name osqueryd --version "$VERSION" -t edge
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
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" updates timestamp
    popd
}

push_to_staging () {
    echo "Running --dryrun push of repository to updates-staging.fleetdm.com..."
    rclone sync --verbose --checksum "$REPOSITORY_DIRECTORY" r2://updates-staging --dry-run
    echo "INFO: This step will push the release to updates-staging.fleetdm.com (staging)..."
    prompt "Check if the above --dry-run looks good."
    # First push the targets/ to avoid sha256 errors on clients.
    rclone sync --verbose --checksum "$REPOSITORY_DIRECTORY/targets/" r2://updates-staging/targets/
    # Then push the rest (json metadata files).
    rclone sync --verbose --checksum "$REPOSITORY_DIRECTORY" r2://updates-staging
    echo "Release has been pushed to staging!"
    echo "NOTE: You might see some clients failing to upgrade due to some sha256 mismatches."
    echo "These temporary failures are expected because it takes some time for caches to be invalidated (these errors should go away after a few minutes minutes)."
}

release_to_production () {
    echo "Running --dryrun server side copy from updates-staging.fleetdm.com to updates.fleetdm.com..."
    rclone sync --verbose --checksum r2://updates-staging r2://updates --dry-run

    echo "WARNING: This step will release to updates.fleetdm.com (production) doing a server copy from updates-staging.fleetdm.com..."
    prompt "Check if the above --dry-run looks good."
    # First push the targets/ to avoid sha256 errors on clients.
    rclone sync --verbose --checksum r2://updates-staging/targets/ r2://updates/targets/
    # Then push the rest (json metadata files).
    rclone sync --verbose --checksum r2://updates-staging r2://updates

    echo "Release has been pushed to production!"
    echo "NOTE: You might see some clients failing to upgrade due to some sha256 mismatches."
    echo "These temporary failures are expected because it takes some time for caches to be invalidated (these errors should go away after a few minutes minutes)."
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
            prompt "To smoke test the release make sure to generate and install fleetd with 'fleetctl package [...] --update-url=https://updates-staging.fleetdm.com --update-interval=1m --orbit-channel=edge --desktop-channel=edge' on Linux amd64, Linux arm64, Windows, and macOS."
        elif [[ $COMPONENT == "osqueryd" ]]; then
            prompt "To smoke test the release make sure to generate and install fleetd with 'fleetctl package [...] --update-url=https://updates-staging.fleetdm.com --osqueryd-channel=edge --update-interval=1m' on Linux amd64, Linux arm64, Windows, and macOS."
        fi
    elif [[ $ACTION == "promote-edge-to-stable" ]]; then
        prompt "To smoke test the release make sure to generate and install fleetd with 'fleetctl package [...] --update-url=https://updates-staging.fleetdm.com --update-interval=1m' on Linux amd64, Linux arm64, Windows, and macOS."
    elif [[ $ACTION == "update-timestamp" ]]; then
        :
    elif [[ $ACTION == "release-to-production" ]]; then
        prompt "To smoke test the release make sure to generate and install fleetd with on Linux amd64, Linux arm64, Windows, and macOS. Use 'fleetctl package [...] --update-interval=1m --orbit-channel=edge --desktop-channel=edge' if you are releasing fleetd to 'edge' or 'fleetctl package [...] --update-interval=1m --osqueryd-channel=edge' if you are releasing osquery to 'edge'."
    elif [[ $ACTION == "create-fleetd-release-pr" ]]; then
        :
    else
        echo "Unsupported action: $ACTION"
        exit 1
    fi
}

fleetctl_version_check () {
    echo "Using '$GIT_REPOSITORY_DIRECTORY/build/fleetctl'"
    "$GIT_REPOSITORY_DIRECTORY/build/fleetctl" --version
    prompt "Make sure the fleetctl executable and version are correct."
}

print_reminder

if [[ $ACTION == "release-to-edge" ]]; then
    trap clean_up EXIT
    setup
    fleetctl_version_check
    pull_from_staging
    release_to_edge
    push_to_staging
elif [[ $ACTION == "promote-edge-to-stable" ]]; then
    trap clean_up EXIT
    setup
    fleetctl_version_check
    pull_from_staging
    promote_edge_to_stable
    push_to_staging
elif [[ $ACTION == "update-timestamp" ]]; then
    trap clean_up EXIT
    setup
    fleetctl_version_check
    pull_from_staging
    update_timestamp
    push_to_staging
elif [[ $ACTION == "release-to-production" ]]; then
    release_to_production
elif [[ $ACTION == "create-fleetd-release-pr" ]]; then
    create_fleetd_release_pr
else
    echo "Unsupported action: $ACTION"
    exit 1
fi

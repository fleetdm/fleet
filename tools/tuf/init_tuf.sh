#!/bin/bash

export FLEET_ROOT_PASSPHRASE=p4ssphr4s3
export FLEET_TARGETS_PASSPHRASE=p4ssphr4s3
export FLEET_SNAPSHOT_PASSPHRASE=p4ssphr4s3
export FLEET_TIMESTAMP_PASSPHRASE=p4ssphr4s3

export TUF_PATH=test_tuf

function create_repository() {
  ./build/fleetctl updates init --path $TUF_PATH

  for system in macos-app macos linux windows; do

    if [[ $system == "macos-app" ]]; then
      # TODO(lucas): Implement code in this branch in a fleetctl command.
      curl -L https://pkg.osquery.io/darwin/osquery-5.1.0.pkg --output $TUF_PATH/tmp/osquery-5.1.0.pkg
      rm -rf $TUF_PATH/tmp/osquery_pkg_expanded
      pkgutil --expand $TUF_PATH/tmp/osquery-5.1.0.pkg $TUF_PATH/tmp/osquery_pkg_expanded
      rm -rf $TUF_PATH/tmp/osquery_pkg_payload_expanded
      mkdir $TUF_PATH/tmp/osquery_pkg_payload_expanded
      tar xf $TUF_PATH/tmp/osquery_pkg_expanded/Payload --directory $TUF_PATH/tmp/osquery_pkg_payload_expanded
      osqueryd_path="$TUF_PATH/tmp/osquery.app.tar.gz"
      tar cf $osqueryd_path $TUF_PATH/tmp/osquery_pkg_payload_expanded/opt/osquery/lib/osquery.app
    else
      # Use latest stable version of osqueryd from our TUF server.
      osqueryd="osqueryd"
      if [[ $system == "windows" ]]; then
        osqueryd="$osqueryd.exe"
      fi
      osqueryd_path="$TUF_PATH/tmp/$osqueryd"
      curl https://tuf.fleetctl.com/targets/osqueryd/$system/stable/$osqueryd --output $osqueryd_path
    fi

    ./build/fleetctl updates add \
      --path $TUF_PATH \
      --target $osqueryd_path \
      --platform $system \
      --name osqueryd \
      --version 42.0.0 -t 42.0 -t 42 -t stable

    rm $osqueryd_path

    goose_value="$system"
    if [[ $system == "macos" || $system == "macos-app" ]]; then
      goose_value="darwin"
    fi
    orbit_target=orbit-$system
    if [[ $system == "windows" ]]; then
      orbit_target="${orbit_target}.exe"
    fi

    # Compile the latest version of orbit from source.
    GOOS=$goose_value go build -o $orbit_target ./orbit/cmd/orbit
    ./build/fleetctl updates add \
      --path $TUF_PATH \
      --target $orbit_target \
      --platform $system \
      --name orbit \
      --version 42.0.0 -t 42.0 -t 42 -t stable
  done
}


if [ ! -d "$TUF_PATH/repository" ]; then
  mkdir -p $TUF_PATH
  mkdir -p $TUF_PATH/tmp
  create_repository
fi

root_keys=$(./build/fleetctl updates roots --path $TUF_PATH)

echo "#########"
echo "Set the following options in 'fleetctl package':"
echo "--update-roots='$root_keys' --update-url=http://localhost:8081"
echo "#########"

echo "Running TUF server..."
go run ./tools/file-server 8081 "${TUF_PATH}/repository"

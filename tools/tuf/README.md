# Testing TUF

Scripts in this directory aim to ease the testing of Orbit and the
[TUF](https://theupdateframework.io/) system.

WARNING: All of these scripts are for testing only, they are not safe for production use.

# Init

To initialize and run a local TUF server, run the `init_tuf.sh` script from the repository root directory:
```sh
./tools/tuf/init_tuf.sh
```

# Add new updates

To add new updates (osqueryd or orbit), use `push_target.sh`.

E.g. to add a new version of `orbit` for Windows:
```sh
# Compile a new version of Orbit:
GOOS=windows GOARCH=amd64 go build -o orbit-windows.exe ./orbit/cmd/orbit

# Push the compiled Orbit as a new version:
./tools/tuf/push_target.sh windows orbit orbit-windows.exe 43
```

E.g. to add a new version of `osqueryd` for macOS:
```sh
# Download some version from our TUF server:
curl --output osqueryd https://tuf.fleetctl.com/targets/osqueryd/macos/5.0.1/osqueryd

# Push the osqueryd target as a new version:
./tools/tuf/push_target.sh macos osqueryd osqueryd 43
```
# Run locally built Fleetd 
(MacOS)


### Run fleet server (and the released Fleetd)
In order to run a local agent (Fleetd + osquery) the first step is to run the Fleet server locally.
Follow this document which will run it together with the released agent.
https://fleetdm.com/docs/contributing/building-fleet

### Modify the Fleetd code as needed

### Build and run locally
In order to use a local version we need to create a local TUF service that will point the installer to take the local Fleetd (instead of the official one).
More details on TUF testing is here:
https://github.com/fleetdm/fleet/tree/main/tools/tuf/test


### MacOS - Prepare a script file with this content. Call it my_build.sh
```sh
SYSTEMS="macos" \
PKG_FLEET_URL=https://localhost:8080 \
PKG_TUF_URL=http://localhost:8081 \
GENERATE_PKG=1 \
ENROLL_SECRET=<REPLACE WITH REAL SECRET KEY> \
FLEET_DESKTOP=1 \
USE_FLEET_SERVER_CERTIFICATE=1 \
./tools/tuf/test/main.sh
```

### Get a real secret key

Go to your local Fleet desktop:
https://localhost:8080/hosts/manage/?order_key=display_name&order_direction=asc
Get the secret key by clicking the __Manage Enroll Secret__

Put the real key here: ```ENROLL_SECRET=<REPLACE WITH REAL SECRET KEY>```

### Remove previous local TUF
If you already have a local TUF running, remove it.

```sh
rm -rf test_tuf
```

### Run the local build
chmod +x my_build.sh
./my_build.sh

### What your build does now
- Download OSQ from github
- Build Fleetd from local src code
- Build fleet desktop from local src code
- Push these three things to the local TUF repository
- Create a local file server to serve the local TUF repository
- Run fleetctl package but instead of the official TUF, it fetches the target from the local TUF
- → the end result is the installer located in ```/Your-Repo-Folder/fleet/fleet-osquery.pkg```

### Install it
Double-click this pkg file and install the local Fleetd.

### Run osquery directly from the Orbit shell
```sudo orbit shell```


<meta name="pageOrderInSection" value="100">
<meta name="description" value="Learn how to build and run Fleetd with modified code.">

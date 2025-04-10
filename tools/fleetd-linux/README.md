# fleetd-linux

This directory contains scripts to build and run Docker Linux images with fleetd installed on them that connect to a Fleet instance running on the host (similar to [tools/osquery](../osquery/)).

PS: In the future, we could push these images to Docker Hub and include some of them in `fleetctl preview` (to allow demoing script execution on Linux hosts).

## Build fleetd docker images

To build all docker images run:
```sh
./tools/fleetd-linux/build-all.sh
```

## Run fleetd containers

To run all fleetd docker images and enroll them to your local Fleet instance, run:
```sh
ENROLL_SECRET=<...> docker compose -f ./tools/fleetd-linux/docker-compose.yml up
```
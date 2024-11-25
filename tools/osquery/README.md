# Configs and Tools for Testing Fleet

The files in this directory are intended to assist with Fleet development.

- `docker-compose.yml`: This docker-compose file helps with starting `osqueryd` instances for testing Fleet. More on this [below](#testing-with-containerized-osqueryd).

- `example_osquery.conf`: An example osquery config file that sets up basic configuration for distributed queries.

- `example_osquery.flags`: An example osquery flagfile setting the config options that must be loaded before the full JSON config.

- `fleet.crt` & `fleet.key`: Self-signed SSL certificate & key useful for testing locally with `osqueryd`. Works with the domain `host.docker.internal` (exposed within docker containers as the host's IP). Should **never** be used in production.

## Testing with containerized osqueryd

Using Docker enables us to rapidly spin up and down pre-configured `osqueryd` instances for testing Fleet. Currently we have container images for Ubuntu14 and Centos7 osquery installations.

### Setup

Docker and docker-compose are the only dependencies. The necessary container images will be pulled from Docker Cloud on first run.

Set the environment variable `ENROLL_SECRET` to the value of your Fleet enroll secret (available on the manage hosts page, or via `fleetctl get enroll-secret`).

(Optionally) Set `FLEET_SERVER` if you want to connect to a fleet server
besides `host.docker.internal:8080`.

### Running osqueryd

The osqueryd instances are configured to use the TLS plugins at `host.docker.internal:8080`. Using the `example_osquery.flags` in this directory should configure Fleet with the appropriate settings for these `osqueryd` containers to connect.

To start one instance each of Centos 6, Centos 7, Ubuntu 14, and Ubuntu 16
`osqueryd`, use:

```
docker-compose up
```

Linux users should use the overrides (which add DNS entries for
`host.docker.internal` based on the `DOCKER_HOST` env var):

```
docker-compose -f docker-compose.yml -f docker-compose.linux-overrides.yml up
```

The logs will be displayed on the host shell. Note that `docker-compose up` will reuse containers (so the state of `osqueryd` will be maintained across calls). To remove the containers and start from a fresh state on the next call to `up`, use:

```
docker-compose rm
```

If you want to only start one instance of `osqueryd`, use:

```
docker-compose run ubuntu14-osquery
```

or

```
docker-compose run centos7-osquery
```

Note that `docker-compose run` does not save state between calls.

This system can also be used to start many instances of osqueryd running in containers on the same host:

```
docker-compose up -d && docker-compose up --scale ubuntu14-osquery=20
```

To stop the containers when running in detached mode like this, use:

```
docker-compose stop
```

And to delete the containers when wanting a fresh state, or when finished testing, use:

```
docker-compose rm
```

We have had no trouble running up to 100 containerized osqueryd instances on a single processor core and about 1GB of RAM.

### Generating an osqueryd core file

The docker containers are configured to allow core files to be generated if osqueryd
crashes for some reason. You can attach to the container hosting the errant osqueryd
instance, install gdb and use it to read the core file to find out where the crash
occurred. The other scenario where you might find a core dump useful is if osqueryd
stops responding. In this case you can generate a core dump using the following instructions.

1. Open a shell session on a container

```
docker exec -t -i <container id> /bin/bash
```

2. Find the process ID of osqueryd

```
ps aux
```

There will be two osqueryd processes, you'll probably be interested in the child process (the one with the higher pid)

3. Send a signal to the process to core dump

```
kill -3 <pid>
```

The core file should be in your current working directory on the container.

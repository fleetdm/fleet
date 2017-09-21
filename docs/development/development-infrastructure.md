Development Infrastructure
==========================

## Starting the local development environment

To set up a canonical development environment via docker, run the following from the root of the repository:

```
docker-compose up
```

This requires that you have docker installed. At this point in time, automatic configuration tools are not included with this project.

#### Stopping the local development environment

If you'd like to shut down the virtual infrastructure created by docker, run the following from the root of the repository:

```
docker-compose down
```

#### Setting up the database tables

Once you `docker-compose up` and are running the databases, you can build the code and run the following command to create the database tables:

```
kolide prepare db
```

## Running Fleet using Docker development infrastructure

To start the Fleet server backed by the Docker development infrasturcture, run the Fleet binary as follows:

```
kolide serve
```

By default, Fleet will try to connect to servers running on default ports on localhost.

If you're using Docker via [Docker Toolbox](https://www.docker.com/products/docker-toolbox), you may have to modify the default values use the output of `docker-machine ip` instead of `localhost`. There is an example configuration file included in this repository to make this process easier for you.  Use the `--config` flag of the Fleet binary to specify the path to your config. See `kolide --help` for more options.

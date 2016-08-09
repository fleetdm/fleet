# Kolide

[![CircleCI](https://circleci.com/gh/kolide/kolide-ose.svg?style=svg&circle-token=2573c239b7f18967040d2dec95ca5f71cfc90693)](https://circleci.com/gh/kolide/kolide-ose)

## Building

To build the code, run the following from the root of the repository:

```
go build -o kolide
```

## Testing

To run the application's tests, run the following from the root of the
repository:

```
go test
```

Or if you using the Docker development environment run:

```
docker-compose app exec go test
```

## Development Environment

To set up a canonical development environment via docker,
run the following from the root of the repository:

```
docker-compose up
```

Once completed, you can access the application at `https://<your-docker-ip>:8080`
where `your-docker-ip` is localhost in most native docker installations.

This requires that you have docker installed. At this point in time,
automatic configuration tools are not included with this project.

If you'd like to shut down the virtual infrastructure created by docker, run
the following from the root of the repository:

```
docker-compose down
```

Once you `docker-compose up` and are running the databases, you can re-build
the code with:

```
docker-compose exec app go build -o kolide
```

and then run the following command to create the database tables:

```
docker-compose exec app ./kolide prepare-db
```

## Docker Deployment
This repository comes with a simple Dockerfile. You can use this to easily
deploy Kolide in any infrastructure context that can consume a docker image
(heroku, kubernetes, rancher, etc).

To build the image locally, run:

```
docker build --rm -t kolide .
```

To run the image locally, simply run:

```
docker run -t -p 8080:8080 kolide
```

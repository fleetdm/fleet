# Kolide

## Building

To build the code, run the following from the root of the repository:

```
go build
```

## Testing

To run the application's tests, run the following from the root of the repository:

```
go test
```

## Development Environment

To set up the development environment via docker, run the following frmo the root of the repository:

```
docker-compose up
```

Obviouly this requires that you have docker installed. At this point in time, automatic configuration tools are not included with this project.

If you'd like to shut down the virtual infrastructure created by docker, run the following from the root of the repository:

```
docker-compose down
```

Once you `docker-compose up` and are running the databases, build the code and run the following command to create the database tables:

```
kolide prepare-db
```
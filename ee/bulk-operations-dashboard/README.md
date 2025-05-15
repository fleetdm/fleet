# Bulk operations dashboard


A dashboard to easily manage profiles and scripts across multiple teams on a Fleet instance.


## Dependencies

- A datastore, this app was built using Postgres, but you can use a database of your choice.

- A Redis database - For session storage.


## Configuration

This app has two required custom configuration values:

- `sails.config.custom.fleetBaseUrl`: The full URL of your Fleet instance. (e.g., https://fleet.example.com)
- `sails.config.custom.fleetApiToken`: An API token for an API-only user on your Fleet instance.

### Required configuration for software features

If you are using this app to manage software across multiple teams on a Fleet instance, five additional configuration values are required:

- `sails.config.uploads.bucket` The name of an AWS s3 bucket where unassigned software installers will be stored.
- `sails.config.uploads.secret` The secret for the S3 bucket where unassigned software installers will be stored.
- `sails.config.uploads.region` The region the AWS S3 bucket is located.
- `sails.config.uploads.bucketWithPostfix`:  The name of the s3 bucket with the directory that the software installers are stored in on appended to it. If the files will be stored in the root directory of the bucket, this value should be identical to the `sails.config.uploads.bucket` value
- `sails.config.uploads.prefixForFileDeletion`: The directory path in the S3 bucket where the software installers will be stored. If the installers will be stored in the root directory, then this value can be set to ' '.


## Running the bulk operations dashboard with Docker.

To run a local bulk operations dashboard with docker, you can follow these instructions.

1. Clone this repo
2. Update the following ENV variables `ee/bulk-operations-dashboard/docker-compose.yml` file:

  1. `sails_custom__fleetBaseUrl`: The full URL of your Fleet instance. (e.g., https://fleet.example.com)

  2. `sails_custom__fleetApiToken`: An API token for an API-only user on your Fleet instance.

  >You can read about how to create an API-only user and get it's token [here](https://fleetdm.com/docs/using-fleet/fleetctl-cli#create-api-only-user)

3. Open the `ee/bulk-operations-dashboard/` folder in your terminal.

4. Run `docker compose up --build` to build the bulk operations dashboard's Docker image.

  > The first time the bulk operations dashboard starts it will Initalize the database aby running the `config/bootstrap.js` script before the server starts.

5. Once the container is done building, the bulk operations dashboard will be available at http://localhost:1337

  > You can login with the default admin login:
  >
  >- Email address: `admin@example.com`
  >
  >- Password: `abc123`

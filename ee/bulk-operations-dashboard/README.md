# Bulk operations dashboard


A dashboard to easily manage profiles and scripts across multiple teams on a Fleet instance.


## Dependencies

- A datastore, this app was built using Postgres, but you can use a database of your choice.


## Configuration

This app has two required custom configuration values:

- `sails.config.custom.fleetBaseUrl`: The full URL of your Fleet instance. (e.g., https://fleet.example.com)

- `sails.config.custom.fleetApiToken`: An API token for an API-only user on your Fleet instance.



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

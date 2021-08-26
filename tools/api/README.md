Using `curl` and `jq` to interact with the fleet API.

First, create a `env` file with the following contents:

```
export SERVER_URL=https://localhost:8080 # your fleet server url and port
export CURL_FLAGS='-k -s' # set insecure flag
export TOKEN=eyJhbGciOi... # your login token
```

Next set the `FLEET_ENV_PATH` to point to the `env` file. This will let the scripts in the `fleet/` folder source the env file.

# Examples

```
export FLEET_ENV_PATH=/Users/victor/fleet_env

# get my user info
./tools/api/fleet/me
{
  "user": {
    "created_at": "2018-04-10T02:07:46Z",
    "updated_at": "2018-04-10T02:07:46Z",
    "id": 1,
    "name": "admin",
    "email": "admin@acme.co",
    "admin": true,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  }
}

# list queries
./tools/api/fleet/queries/list
{
  "queries": []
}

# use jq to filter a specific query and get the id
./tools/api/fleet/queries/list | jq '.queries[]|select(.name == "osquery_info")|.id'
2

# create a query
./tools/api/fleet/queries/create 'system_info' 'select * from system_info;'
{
  "query": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 4,
    "name": "system_info",
    "description": "",
    "query": "select * from system_info;",
    "saved": true,
    "author_id": 1,
    "author_name": "admin",
    "packs": []
  }
}

# add query with id=4 to pack with id=2
./tools/api/fleet/schedule/add_query_to_pack 2 4

# get scheduled queries in a pack
./tools/api/fleet/packs/scheduled 2 | jq '.scheduled[]|{"name": .name, "schedule_id": .id, "query_id": .query_id}'
```

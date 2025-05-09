[Back to top](./README.md)
# Fleetctl

CLI interface to using the API

Distributed via NPM and Docker

## Commands

### api

This is the swiss army knife of fleetctl and let's you use the fleet api raw from the command line
with your already logged in contexts. Largely modeled after `gh api` this allows you to test new
api's you are developing or create scripts around api functionality that might not have full
fleetctl support yet.

Example usage

```
fleetctl api /scripts
{
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  },
  "scripts": [
    {
      "id": 1,
      "team_id": null,
      "name": "check_boot.sh",
      "created_at": "2024-09-05T01:41:48Z",
      "updated_at": "2024-09-05T01:41:48Z"
    },
    {
      "id": 2,
      "team_id": null,
      "name": "hello_world.ps1",
      "created_at": "2024-09-05T01:41:55Z",
      "updated_at": "2024-09-05T01:41:55Z"
    }
  ]
}
```

By default it will do http GET but you can modify the method, and set headers and parameters

```
   -F value, --field value [ -F value, --field value ]    Add a typed parameter in key=value format
   -H value, --header value [ -H value, --header value ]  Add a HTTP request header in key:value format
   -X value                                               The HTTP method for the request (default: "GET")
```

### apply
### config
### convert
### debug
### delete
### flags
### generate
### get
### gitops
### goquery
### hosts
### kill_process
### login
### logout
### mdm
### package
### preview
### query
### scripts
### setup
### trigger
### user
### vulnerability-data-stream
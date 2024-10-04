# REST API

Use the Fleet APIs to automate Fleet.

- [Activities](https://fleetdm.com/docs/rest-api/activities)
- [Authentication](https://fleetdm.com/docs/rest-api/authentication)
- [Commands](https://fleetdm.com/docs/rest-api/commands)
- [Debug](https://fleetdm.com/docs/rest-api/debug)
- [File carving](https://fleetdm.com/docs/rest-api/file-carving)
- [Fleet configuration](https://fleetdm.com/docs/rest-api/fleet-configuration)
- [Hosts](https://fleetdm.com/docs/rest-api/hosts)
- [Integrations](https://fleetdm.com/docs/rest-api/integrations)
- [Labels](https://fleetdm.com/docs/rest-api/labels)
- [OS Settings](https://fleetdm.com/docs/rest-api/os-settings)
- [Policies](https://fleetdm.com/docs/rest-api/policies)
- [Queries](https://fleetdm.com/docs/rest-api/queries)
- [Scripts](https://fleetdm.com/docs/rest-api/scripts)
- [Setup experience](https://fleetdm.com/docs/rest-api/setup-experience)
- [Software](https://fleetdm.com/docs/rest-api/software)
- [Targets](https://fleetdm.com/docs/rest-api/targets)
- [Teams](https://fleetdm.com/docs/rest-api/teams)
- [Users](https://fleetdm.com/docs/rest-api/users)
- [Vulnerabilities](https://fleetdm.com/docs/rest-api/vulnerabilities)

## API errors

Fleet returns API errors as a JSON document with the following fields:
- `message`: This field contains the kind of error (bad request error, authorization error, etc.).
- `errors`: List of errors with `name` and `reason` keys.
- `uuid`: Unique identifier for the error. This identifier can be matched to Fleet logs which might contain more information about the cause of the error.

Sample of an error when trying to send an empty body on a request that expects a JSON body:
```sh
$ curl -k -H "Authorization: Bearer $TOKEN" -H 'Content-Type:application/json' "https://localhost:8080/api/v1/fleet/sso" -d ''
```
Response:
```json
{
  "message": "Bad request",
  "errors": [
    {
      "name": "base",
      "reason": "Expected JSON Body"
    }
  ],
  "uuid": "c0532a64-bec2-4cf9-aa37-96fe47ead814"
}
```

---


<meta name="description" value="Documentation for Fleet's REST API. See example requests and responses for each API endpoint.">
<meta name="pageOrderInSection" value="1">

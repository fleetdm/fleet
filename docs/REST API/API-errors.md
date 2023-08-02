# API errors

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


<meta name="description" value="Read about how Fleet's REST API returns errors.">
<meta name="pageOrderInSection" value="1900">
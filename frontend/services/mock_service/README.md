# How to use mock service

The mock service implements the `sendRequest` interface and enables you to more easily develop
frontend features that make use of spec API endpoints without waiting for backend development to be
completed. 

You configure the mock service by mapping the API request paths and expected JSON responses 
based on the API specification for your feature. Then, you simply import the mock service 
in place of `sendRequest` when you build out the `/services/entities` methods for your feature. 

The mock service simulates async network requests and responses for the API. The mock service tries 
to match incoming request URLs against the paths declared in `REQUEST_RESPONSE_MAPPINGS`. 
If a match is found, the mock service returns the expected JSON response with `Promise.resolve`. 
If no match is found, the mock service returns an error with `Promise.reject`.
 
## Importing the mock service

To use the mock service in development, import the following in the `/services/entities` file 
for your API service.
```js
import { sendRequest } from "services/mock_service/service/service";
```
When the real API is ready, swap out the mock service import with the normal `sendRequest` import.
```js
import { sendRequest } from "services";
``` 

## Configuring the mock service

Configuration consists of two files: `config` and `responses`. 

### Responses file
Declare your static JSON responses as constants in the `responses` file inside the `./mocks` folder.
Each JSON response should be given its own unique name and added to the default export for the file. 
These responses will be imported as `STATIC` in the `config` file where you will map the responses
to the the request paths for your API service.

### Config file
Declare your endpoint and each of your API request paths to its expected JSON response in the
`config` file inside the `./mocks` folder. 

Set the `DELAY` constant (in milliseconds) if you want to simulate a delayed API response.

Set the `ENDPOINT` constant to the base route for your API endpoint (for example, `/latest/fleet`). 

Use the `REQUEST_RESPONSE_MAPPINGS` dictionary to declare your request-responses mappings. For example,
here's how you might configure the `GET hosts/manage` and `GET hosts/count`endpoints:
```js
const REQUEST_RESPONSE_MAPPINGS = {
    GET: {
        "/hosts/manage": STATIC.MANAGE,
        "/hosts/count": STATIC.COUNT
    }
};
```

You can declare different responses for specific route parameters or query parameters. Alteratively,
you can use wildcards if you don't particularly care what parameter value is contained in the request. 
The example below shows how you might use wildcards or specific values as the `id` route parameter 
to get a host's device mapping by host id and how you might use wildcards or specific values 
for the `team_id` query parameter to get hosts filtered by team id.
```js
const REQUEST_RESPONSE_MAPPINGS = {
    GET: {
        "/hosts/1/device_mapping": STATIC.HOST_1_DEVICE_MAPPING, // specific route param value
        "/hosts/:id/device_mapping": STATIC.HOST_ID_DEVICE_MAPPING, // wildcard route param value
        "/hosts/manage?team_id=1": STATIC.HOSTS_TEAM_1, // specific query param value
        "/hosts/manage?team_id={id}": STATIC.HOSTS_TEAM_ID // wildcard query param value

    }
};
```
For purposes of illustration, the example above uses ":" as well as "{" and "}" for wildcard
characters. You can set the `WILDCARDS` constant in the `config` file to define any number 
of wildcards to suit the conventions of the API spec for your feature. 

The mock service evaluates URLs part-by-part. Each URL is split at the "?" character. If more than
one "?" is present, the mock service throws an error. If this happens in the context of an API request, the
mock service returns with `Promise.reject`. Assuming only one "?" is present, the first half is
split into parts at each "/" and the second half is split into parts at each "&". The substring parts are
evaluated for matching purposes in the order they appeared in the URL string. If no "?" is present,
the URL will only be split by "/".

The presence of one or more wildcard characters anywhere in a substring part of the URL path
declared in `REQUEST_RESPONSE_MAPPINGS` triggers a match against the corresponding part of the
request URL. More precise handling of wildcard parameters is something that may be added in the
future. In the meantime, you should take care that the path you declare in `REQUEST_RESPONSE_MAPPINGS` 
follows the order of the params in the request made by your `/services/entities` methods. 

# Examples
Example `config` and `response` files are included in `./examples`. Please be sure to copy these
files into `./mocks` before making any changes.
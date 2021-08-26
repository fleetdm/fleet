# Kolide Request Mocks

Request mocks are used to intercept API requests when running tests. Requests
are mocked to simulate valid and invalid requests. The naming convention is
similar to the [API client entity CRUD methods](../../fleet/README.md).

## Using Mocks

```js
// import the mocks you want in the test file
import queryMocks from 'test/mocks/query_mocks';

// mock the API request before making the API call
queryMocks.load.valid(bearerToken, queryID); // valid request
queryMocks.load.invalid(bearerToken, queryID); // invalid request
```

Each entity with mocked requests has a dedicated file in this directory
containing the mocks for the entity, such as `queryMocks` in the example above. If requests
need to be mocked for multiple entities, consider importing all mocks:

```js
import mocks from 'test/mocks';

mocks.queries.load.valid(bearerToken, queryID);
mocks.packs.create.valid(bearerToken, params);
```

## Creating Mocks

Mocks are created using the [`createRequestMock`](./create_request_mock.js) function.

The `createRequestMock` function returns a mocked request using the [nock](https://github.com/node-nock/nock) npm package.

Example:

```js
// in /frontend/test/mocks/query_mocks.js
import createRequestMock from 'test/mocks/create_request_mock';
import { queryStub } from 'test/stubs';

const queryMocks = {
  load: {
    valid: (bearerToken, queryID) => {
      return createRequestMock({
        bearerToken,
        endpoint: `/api/v1/fleet/queries/${queryID}`,
        method: 'get',
        response: { query: { ...queryStub, id: queryID } },
        responseStatus: 200,
      });
    },
  },
}

export default queryMocks;
```

`createRequestMock` takes an options hash with the following options:

`bearerToken`

* Type: String
* Required?: False
* Default: None
* Purpose: Specifying the bearer token sets the Authorization header of the
  request and is often used when mocking authorized requests to the API.

`endpoint`

* Type: String
* Required?: True
* Default: None
* Purpose: The required endpoint option is the relative pathname of the request.

`method`

* Type: String (`get` | `post` | `patch` | `delete`)
* Required?: True
* Default: None
* Purpose: This string is the lower-cased request method. Options are `get`,
  `post`, `patch`, and `delete`.

`params`

* Type: Object
* Required?: False
* Default: None
* Purpose: This JS Object is for the parameters sent with a request. If the
  parameters are URL parameters, such as in a GET request, add the parameters to
the `endpoint` option.

`response`

* Type: Object
* Required?: True
* Default: None
* Purpose: This JS Object represents the response from the API

`responseStatus`

* Type: Number
* Required?: False
* Default: 200
* Purpose: This value is used for the response status of the API call.

### Examples

[API Request](../../fleet/entities/packs.tests.js#L16-L30)
* The mocked request is saved as a variable in order to assert that the request
  is made


[Component Test](../../components/forms/fields/SelectTargetsDropdown/SelectTargetsDropdown.tests.jsx#L35-L40)
* The request is not saved but we want to prevent attempting to make an API request.
* There is no API to hit in tests so attempting to make an API call with result
  in warnings in the test output.

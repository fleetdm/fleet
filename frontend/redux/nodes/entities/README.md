# Fleet Redux Entities

Redux entities are all configured similarly in Fleet. They start with the same initial
state, and contain the same default actions (each action namespaced to the
entity name). Some entities extend the default configuration but for the most
part entities are configured as follows:

## Entity Configuration

Entities are configured from the [base configuration
class](./base/config.js).
The base configuration takes an options hash with the following attributes.

`createFunc`: Function

- The function that calls the API to create an entity.

`destroyFunc`: Function

- The function that calls the API to destroy an entity.

`entityName`: String

- The name of the entity (ie 'users').

`loadAllFunc`: Function

- The function that calls the API to load all entities.

`loadFunc`: Function

- The function that calls the API to load an individual entity or subset of
  entities.

`parseApiResponseFunc`: Function

- A function that is used to parse the response from the API client (optional).

`parseEntityFunc`: Function

- A function that is used to parse an entity returned from the API (optional).
  This can be useful for formatting entity attributes or adding new attributes.

`schema`: Schema

- This is a [normalizr schema](https://github.com/paularmstrong/normalizr/blob/master/docs/api.md#schema) used for structuring the entities in state.

`updateFunc`: Function

- The function that calls the API to update an entity.

### Example Entity Configuration

```js
// in redux/nodes/entities/packs/config.js
import API from fleet;
import ReduxConfig from 'redux/nodes/entities/base/config';
import schemas from 'redux/nodes/entities/base/schemas';

const config = new ReduxConfig({
  createFunc: API.packs.create,
  destroyFunc: API.packs.destroy,
  entityName: 'packs',
  loadAllFunc: API.packs.loadAll,
  loadFunc: API.packs.load,
  schema: schemas.PACKS,
  updateFunc: API.packs.update,
});

export default config;
```

An instance of the `ReduxConfig` class exposes 2 computed properties, `actions`,
and `reducer`.

## Actions

The default actions are as follows, namespaced to the config's `entityName`
string:

`clearErrors`

- Clears the entity's errors in state

`loadAll`

- A [thunk action](https://github.com/gaearon/redux-thunk) that does the following:

1. dispatches the `loadRequest` action type to alert the app that an API call is being made.
2. calls the `loadAllFunc` to call the API.
3. if successful, dispatches the `loadAllSuccess` action type with the parsed API entities.
4. if unsuccessful, dispatches the `loadFailure` action type with the formatted API errors.

`silentLoadAll`

- A thunk action that does the same thing as the `loadAll` action without the first step of alerting the app that an API call is being made.
- This action is useful when we have a page where in some cases we want to
  display a loading spinner during API calls and in some cases we do not.

`successAction`

- A function that accepts 2 parameters, an API response and a thunk. The
  `successAction` does the work of parsing the API response to get the formatted
  entities and then calls the thunk parameter, passing the thunk the formatted
  entities. It is used within the configuration class to format API responses and
  call success actions with the formatted responses.
- This function can be helpful when extending the base set of actions. Otherwise
  it can be ignored.

`create`

- A thunk action that does the following:

1. dispatches the `createRequest` action to alert the app that an API call is being made.
2. calls the `createFunc` to call the API.
3. if successful, dispatches the `createSuccess` action with the parsed API entities.
4. if unsuccessful, dispatches the `createFailure` action with the formatted API errors.

`silentCreate`

- A thunk action that does the same thing as the `create` action without the first step of alerting the app that an API call is being made.
- This action is useful when we have a page where in some cases we want to
  display a loading spinner during API calls and in some cases we do not.

`createRequest`

- An action with a namespaced action type, such as `users_CREATE_REQUEST` to
  alert the app that an API call is being made for a specific entity type.

`createSuccess`

- An action with a namespaced action type, such as `users_CREATE_SUCCESS` which
  the reducer uses to add the action payload to the entity's data.

`createFailure`

- An action with a namespaced action type, such as `users_CREATE_FAILURE` which
  the reducer uses to add the action payload to the entity's errors.

`destroy`

- A thunk action that does the following:

1. dispatches the `destroyRequest` action to alert the app that an API call is being made.
2. calls the `destroyFunc` to call the API.
3. if successful, dispatches the `destroySuccess` action.
4. if unsuccessful, dispatches the `destroyFailure` action with the formatted API errors.

`silentDestroy`

- A thunk action that does the same thing as the `destroy` action without the first step of alerting the app that an API call is being made.
- This action is useful when we have a page where in some cases we want to
  display a loading spinner during API calls and in some cases we do not.

`destroyRequest`

- An action with a namespaced action type, such as `users_DESTROY_REQUEST` to
  alert the app that an API call is being made for a specific entity type.

`destroySuccess`

- An action with a namespaced action type, such as `users_DESTROY_SUCCESS` which
  the reducer uses to remove the entity from state.

`destroyFailure`

- An action with a namespaced action type, such as `users_DESTROY_FAILURE` which
  the reducer uses to add the action payload to the entity's errors.

`load`

- A thunk action that does the following:

1. dispatches the `loadRequest` action to alert the app that an API call is being made.
2. calls the `loadFunc` to call the API.
3. if successful, dispatches the `loadSuccess` action with the parsed API entities.
4. if unsuccessful, dispatches the `loadFailure` action with the formatted API errors.

`silentLoad`

- A thunk action that does the same thing as the `load` action without the first step of alerting the app that an API call is being made.
- This action is useful when we have a page where in some cases we want to
  display a loading spinner during API calls and in some cases we do not.

`loadRequest`

- An action with a namespaced action type, such as `users_LOAD_REQUEST` to
  alert the app that an API call is being made for a specific entity type.

`loadSuccess`

- An action with a namespaced action type, such as `users_LOAD_SUCCESS` which
  the reducer uses to add the action payload to the entity's data.

`loadFailure`

- An action with a namespaced action type, such as `users_LOAD_FAILURE` which
  the reducer uses to add the action payload to the entity's errors.

`update`

- A thunk action that does the following:

1. dispatches the `updateRequest` action to alert the app that an API call is being made.
2. calls the `updateFunc` to call the API.
3. if successful, dispatches the `updateSuccess` action with the parsed API entity.
4. if unsuccessful, dispatches the `updateFailure` action with the formatted API errors.

`silentUpdate`

- A thunk action that does the same thing as the `update` action without the first step of alerting the app that an API call is being made.
- This action is useful when we have a page where in some cases we want to
  display a loading spinner during API calls and in some cases we do not.

`updateRequest`

- An action with a namespaced action type, such as `users_UPDATE_REQUEST` to
  alert the app that an API call is being made for a specific entity type.

`updateSuccess`

- An action with a namespaced action type, such as `users_UPDATE_SUCCESS` which
  the reducer uses to replace the entity in state with the action's payload.

`updateFailure`

- An action with a namespaced action type, such as `users_UPDATE_FAILURE` which
  the reducer uses to add the action payload to the entity's errors.

## Reducer

The ReduxConfig reducer configures entities with the same initial state:

```js
{
  loading: false,
  data: {},
  errors: {},
  originalOrder: [],
}
```

Entities are stored in state in the `data` object with a key of the entity ID
and a value of the entity object. For example, the users in state might look
like the following:

```js
{
  loading: false,
  data: {
    10: {
      first_name: 'Mike',
      last_name: 'Stone',
      id: 10,
    },
    14: {
      first_name: 'Kyle',
      last_name: 'Knight',
      id: 14,
    },
  },
  errors: {},
  originalOrder: [14, 10],
}
```

Entity order from the API is also tracked in the `originalOrder` array. This is can be used
for rendering lists of entities in a specific order that is determined by
the API response (e.g. hosts table). In the above example the API sent back
the users in this order.

```js
users: [
  {
    first_name: "Kyle",
    last_name: "Knight",
    id: 14,
  },
  {
    first_name: "Mike",
    last_name: "Stone",
    id: 10,
  },
];
```

Entity errors in state are an object with a key/value pair that corresponds to
an invalid attribute and the error message for that attribute as returned from
the server. For example, the user errors in state might look like the following
after attempting to create a user with invalid data:

```js
{
  loading: false,
  data: {},
  errors: {
    first_name: 'First name cannot be blank',
  },
  originalOrder: [],
}
```

Request actions, such as `createRequest`, `loadRequest`, `updateRequest`, and
`destroyRequest` are handled in the reducer to change the `loading` boolean from
`false` to `true`.

When a request finishes and either the success or failure actions are dispatched
as a result, the reducer sets the `loading` boolean back to `false`.

Success actions include a payload object with a data key. The data for `create`,
`update`, `destroy`, `load`, and `loadAll` success actions are the entity or an array of
entities that are then added, updated, or removed from state.

Failure actions include a payload object with an errors key containing an object
of server errors for the entity. These server errors are added to state in the
entity's `errors` key.

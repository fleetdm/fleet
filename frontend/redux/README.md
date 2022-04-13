# Fleet Redux implementation

Fleet uses [`Redux`](http://redux.js.org) for application state management.
React components themselves can manage state, but Redux makes it easy to share
state throughout the app by being the single source of truth (such as keeping track of the entities returned by the API).

To learn more about Redux visit http://redux.js.org.

## Redux state structure

### Overview

The shape of the application's Redux state is as follows:

```js
{
  app: {
    ...
  },
  auth: {
    ...
  },
  components: {
    ...
  },
  entities: {
    ...
  },
  loadingBar: {
    ...
  },
  notifications: {
    ...
  },
  redirectLocation: {
    ...
  },
  routing: {
    ...
  },
}
```

### App state

App state contains information about the general app setup and information. It
contains a `config` object with data on the user's organization and Fleet
setup.

### Auth state

Auth state contains data on the current user.

### Component state

Component state contains data specific to React components.

### Entities state

The entities state holds data on specific entities such as users, queries,
packs, etc. They follow a similar configuration that can be found [here](./nodes/entities/README.md).

### Notifications state

The notifications state contains data that informs the rendering of flash
messages.

### Redirect location state

The redirect location state contains information about where to redirect a user
after login, specifically when they attempt to access an authenticated route when logged out
and then log in.

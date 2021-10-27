# Fleet Redux Middleware

The Fleet Redux Middleware handles actions before they hit the reducers. The
current middleware does the following:

## [Authentication Middleware](./auth.js)

The authentication middleware handles logging a user in/out and handles logging out a user when the API responds
with an unauthenticated error.

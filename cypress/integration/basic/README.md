# Basic tier tests

These tests should only run when the server is in `basic` tier.

Any test suite in this directory should be surrounded by the following:

```js
if (Cypress.env("FLEET_TIER") === "basic") {
  // test suite here
}
```

FIXME: There must be a better way to do this for all tests in the directory?

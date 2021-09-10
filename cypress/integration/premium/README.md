# Premium tier tests

These tests should only run when the server is in `premium` tier.

To enable the tests:

```sh
export CYPRESS_FLEET_TIER=premium
```

Before running the appropriate `yarn cypress (open|run)` command.

## Filtering

Any test suite in this directory should use the following pattern for filtering:

**FIXME**: There must be a better way to do this for all tests in the directory rather than having to add the check in each file?

```js
if (Cypress.env("FLEET_TIER") === "premium") {
  // test suite here
}
```

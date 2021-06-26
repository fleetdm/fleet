# Kolide Tests

The test directory contains helper functions, request mocks, and entity stubs
for use in test files throughout the application. The test files for components and app functions are located in the same directory as the files they test.

The default export from the test directory includes Helpers, Mocks, and Stubs.

## [Helpers](./helpers.jsx)

```js
import Test from 'test';

const helpers = Test.Helpers;
```

The helpers file includes functions that make certain test actions easy, such as
mounting a connected component, building a mock redux store, and filling in a
form input.

Below are a couple particularly useful test helpers.

### fillInFormInput

This function is useful when the component renders a form that needs to be
filled out. The function takes to parameters, the form input element and the
value to be filled in.

[Example](../components/forms/UserSettingsForm/UserSettingsForm.tests.jsx#L27-L43)

### reduxMockStore

This function is useful for creating a fake redux store. This store will allow
actions to be dispatched, keep a collection of dispatched actions, and hold
state.

[Example using mockStore.dispatch](../components/forms/UserSettingsForm/UserSettingsForm.tests.jsx#L27-L43)

[Example using mockStore.dispatch](../redux/nodes/auth/actions.tests.js#L36-L67)

[Example mounting a connected component and checking dispatched actions](../pages/RegistrationPage/RegistrationPage.tests.jsx#L18-L42)

### connectedComponent

The `connectedComponent` function is useful for mounting connected components in
tests (usually Page components). This helper wraps the component in a
`Provider` component and sets a mock store as the redux store for the connected
component. It takes 2 parameters, the component class and an options hash. The
options has 2 optional keys, `mockStore` and `props`. Keep in mind that when
mounting a connected component `mapStateToProps` will run and the component will
receive the props assigned in the `mapStateToProps` function.

[Example](../pages/RegistrationPage/RegistrationPage.tests.jsx)

[Example](../components/queries/QueryPageWrapper/QueryPageWrapper.tests.jsx)

## [Mocks](./mocks/README.md)

```js
import Test from 'test';

const mocks = Test.Mocks;
```

Documentation on request mocks can be found in the [Kolide Request Mock
Documentation](./mocks/README.md)

## [Stubs](./stubs.ts)

```js
import Test from 'test';

const stubs = Test.Stubs;
```

The Stubs file contains objects that represent entities used in the Kolide
application. These re-usable objects help keep the code DRY.

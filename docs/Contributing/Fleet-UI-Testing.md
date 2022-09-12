# Frontend Testing Strategy & Plan

This document contains the testing strategy and plan for the frontend codebase. The testing
strategy is a high level overview of the **Who**, **What**, **When** and **Why** when it comes to testing.
The testing plan primarily outlines the **How** of testing and covers the different practices and
toolings used in testing.

**Table Of Contents**

- [Testing Strategy](#testing-strategy)
  - [Testing Philosophy](#testing-philosophy)
  - [Who Tests](#who-tests)
  - [What We Test](#what-we-test)
    - [Shared Utilities](#shared-utilities)
    - [UI Building Blocks](#ui-building-blocks-resusable-components-and-hooks)
    - [App Widgets](#app-widgets)
    - [User Journyes](#user-journeys)
- [Testing Plan](#testing-plan)
  - [Types of Test](#types-of-tests)
    - [Testing Cheat Sheet](#testing-cheat-sheet)
    - [Manual Testing](#manual-testing)
    - [Static Analysis](#static-analysis)
    - [Unit Testing](#unit-testing)
      - [Shared Utilities Testing](#shared-utilities-testing)
      - [UI Building Block Testing](#ui-building-block-testing)
      - [Reusable Hook Testing](#reusable-hooks-testing)
    - [Integration Testing](#integration-testing)
      - [App Widgets Testing](#app-widget-testing)
    - [E2E Testing](#e2e-testing)
  - [Tooling](#tooling)
    - [Eslint and Typescript](#eslint-and-typescript)
    - [Jest](#jest)
    - [Testing-Library](#testing-library)
    - [Cypress](#cypress)
  - [More Specific Examples](#more-sepcific-exampels)
    - [Roles and Permissions](#roles-and-permissions)
    - [Mac and Windows Hosts](#mac-and-windows-hosts)
    - [Error States](#error-states)

---

## Testing Strategy

### Testing Philosophy

When we create tests, we keep in mind how an end user will be using this software. This idea influences all other decisions when it comes to testing. Sometimes the end user is a company admin using the fleet UI, or a devops using fleetctl in the terminal, or a developer using a reusable UI component or utility or class; In any case we should first think about the end user when building our testing plan. Testing software from this perspective has many advantages including:

- A focus on functionality and behavior over implementation details. This leads to better maintainability of the testing suite which does not have to change as the implementation changes.
- Gives a clear idea of what type of tests are useful and should be prioritised.
- Gives higher confidence that the software behaves as intended in a real world scenario

### Who Tests

The **developer** is responsible for writing and maintaining tests as part of their work. They can get
help from QA when trying to decide what tests are sufficient for the feature.

### What We Test

We break down our application into different type of systems that we can test in ways that best fit
their use cases.

#### Shared Utilities

Shared utilities are generally simple JS functions used throught the app. This includes code like
utils in `utilites/url` and `utilities/string`.

#### UI Building Blocks (resusable components and hooks)

UI Building blocks is code that is reusable and primarialy props or argument driven. These include
components in the `components` directory like `Radio`, `FlashMessage`, and `Button` or reusable
hooks like `useDeepEffect`.


#### App Widgets

App widgets are larger chunks of the application that are more more specific (therefore not
reused) and are made up the UI building blocks. They tend to have more context and dependencies
within them and therefore are tested differently than the
building blocks. Examples include forms like `LoginForm`, and `ResetPasswordForm` or even some
simpler pages including `Registration Page` or `LoginPage`

#### User Journeys

User Journeys are the flows users take through the application to accomplish their goal. These are
typically the widest and can include navigating through multiple pages or working with multiple app
widgets on a page. This would include things like **Creating a new user or team** or **filtering
hosts by software vulnerabilities or set policies**.

---

## Testing Plan

This section answers how we are testing our code. We cover tools and practices to writing tests. We
like to test the software at different layers (unit, integration, e2e) all of which have their
own usefulness and testing practices.

> NOTE: Architecture plays a huge role in testing practices and tools. In our current landscape
(__as of 31/08/2022__) we do not have the best seperation of concerns between our systems. As a
result, the most reliable way to test our software is as a whole with e2e tests. While this is ok
at some level, we still are missing a large chunk of important testing that would be better written
and maintained at the integration/unit level. We'd like to utilise seperation of concerns more
between our systems as this will allow us to test more in the integration and unit layers which are
quicker to run and generally easier to work with.

### Types of Tests

We use a variety of testing to ensure that our software is working as intended. This includes:

- End-to-end (e2e)
- Integration
- Unit
- Static
- Manual

#### Testing Cheat Sheet

| Systems | Type of Test | Who is the User | Example | Notes |
| ------- | ------------ | --------------- | ------- | ----- |
| Reusuable utilites | unit w/jest | devs | string util, url util | no/few dependencies. function argument based |
| Reusable hooks | unit w/jest & react-hooks lib | devs | useToggleDisplayed Hook | no/few dependencies. function argument based |
| Reusable UI components | unit w/jest & testing-library | devs | Radio, Button, Input components | no/few dependencies. props based |
| App Widgets | Integration or e2e w/testing-library or cypress | end users | Create User form, Reset Password form | Less reusable code with more complex environment setup and dependencies. Depending on the case can be done with integration or e2e. When integration, mock at the backend level, dont mock other UI systems. When e2e, no mocking of other systems. |
| User Journeys | e2e w/cypress | end users | filtering a host by software, creating a team as admin | Full business flows. No/minimal mocking of systems, except for common error states from network. |
| N/A | Manual | N/A | // TODO | Manual testing can be used for all types of code. Examples would be for one offs or states that would require extreamly difficult testing setups. |

#### Manual Testing

There will always be a space for manual testing and we utilise it for testing states that do not
occur very often in the application or are not worth the effort to set up to test for an  unlikely edge case.

#### Static Analysis

This includes typing and linting to quickly ensure proper typings of data flowing through the application and that we are following coding conventions and styling rules. This give us a first of defense against writing buggy code.

#### Unit Testing

We unit test smaller reusable components that have no/few  dependencies within them. These
components are primarily parametric based and require no or minimal mocking to be tested
effectively. They tend to be small building blocks of our application (e.g. reusable UI components,
common utilities, reusable hooks. The end user in this case tends to be other developers so we want
to ensure these components work as expected when used as building blocks.

##### Shared Utilities Testing

We can test these purely with jest and dont have to worry about rendering components with
react-testing-library. Only jest is needed with minimal mocking.

[source here](https://github.com/fleetdm/fleet/blob/main/frontend/utilities/url/url.tests.ts)

```tsx
import { buildQueryStringFromParams } from ".";

describe("url utilites", () => {
  it("creates a query string from a params object", () => {
    const params = {
      query: "test",
      page: 1,
      order: "asc",
      isNew: true,
    };
    expect(buildQueryStringFromParams(params)).toBe(
      "query=test&page=1&order=asc&isNew=true"
    );
  });

  it("filters out undefined values", () => {
    const params = {
      query: undefined,
      page: 1,
      order: "asc",
    };
    expect(buildQueryStringFromParams(params)).toBe("page=1&order=asc");
  });
});
```
##### UI Building Block Testing

We test the component to ensure it works how a developer would want to use it. There is minimal
mocking and we utilize react-testing-library and jest spies to test.

[source here](https://github.com/fleetdm/fleet/blob/main/frontend/components/forms/fields/Radio/Radio.tests.tsx)

```tsx
import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import Radio from "./Radio";

describe("Radio - component", () => {
  it("renders the radio label text from the label prop", () => {
    render(
      <Radio
        checked
        label={"Radio Label"}
        value={"radioValue"}
        id={"test-radio"}
        onChange={noop}
      />
    );

    const labelText = screen.getByText("Radio Label");
    expect(labelText).toBeInTheDocument();
  });

  it("passes the radio input value when checked", async () => {
    const user = userEvent.setup();
    const changeHandlerSpy = jest.fn();

    render(
      <Radio
        label={"Radio Label"}
        value={"radioValue"}
        id={"test-radio"}
        onChange={changeHandlerSpy}
      />
    );

    const radio = screen.getByRole("radio", { name: "Radio Label" });
    await user.click(radio);

    expect(changeHandlerSpy).toHaveBeenCalled();
    expect(changeHandlerSpy).toHaveBeenCalledWith("radioValue");
  });
});
```

##### Reusable Hooks Testing

// TODO

#### Integration Testing

We use integration testing to strike a balance between speed and expense to write our tests. We use
them to test multiple reusable components that come together into less reusable App Widgets. We also try to use
minimal mocking, but can mock at the backend level if required.

##### App widget Testing

This layer of tests is great for testing difficult to obtain states and edge cases as the test setup
is simpler than e2e tests. We highly utilise react-testing-library to interface with these components.

[source here](https://github.com/fleetdm/fleet/blob/main/frontend/components/forms/ResetPasswordForm/ResetPasswordForm.tests.jsx)

```tsx
import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/testingUtils";

import ResetPasswordForm from "./ResetPasswordForm";

describe("ResetPasswordForm - component", () => {
  const newPassword = "password123!";
  const submitSpy = jest.fn();

  it("does not submit the form if the password is invalid", async () => {
    const invalidPassword = "invalid";
    const { user } = renderWithSetup(
      <ResetPasswordForm handleSubmit={submitSpy} />
    );

    await user.type(screen.getByLabelText("New password"), invalidPassword);
    await user.type(screen.getByLabelText("Confirm password"), invalidPassword);
    await user.click(screen.getByRole("button", { name: "Reset password" }));

    const passwordError = screen.getByText(
      "Password must meet the criteria below"
    );
    expect(passwordError).toBeInTheDocument();
    expect(submitSpy).not.toHaveBeenCalled();
  });
});
```

#### E2E testing

Our e2e layer tests all the systems of the software (frontend and backend) together to
ensure the application works as intended. To support this we do not often mock API responses at this
layer of testing (with an exception for mocking network error responses). We want to test the
software as an actual user would at this level.

[source here](https://github.com/fleetdm/fleet/blob/main/cypress/integration/all/app/labelflow.spec.ts)

```ts
describe("Labels flow", () => {
  before(() => {
    // ...setup
  });
  after(() => {
    // ...teardown
  });

  describe("Manage hosts page", () => {
    beforeEach(() => {
      // ...setup
    });
    it("creates a custom label", () => {
      cy.getAttached(".label-filter-select__control").click();
      cy.findByRole("button", { name: /add label/i }).click();
      cy.getAttached(".ace_content").type(
        "{selectall}{backspace}SELECT * FROM users;"
      );
      cy.findByLabelText(/name/i).click().type("Show all MAC users");
      cy.findByLabelText(/description/i)
        .click()
        .type("Select all MAC users.");
      cy.getAttached(".label-form__form-field--platform > .Select").click();
      cy.getAttached(".Select-menu-outer").within(() => {
        cy.findByText(/macOS/i).click();
      });
      cy.findByRole("button", { name: /save label/i }).click();
      cy.findByText(/label created/i).should("exist");
    });
  });
});
```


### Tooling

Here is a quick reference of the current tooling we are using at each layer of testing.

<img src="https://miro.medium.com/max/1400/1*iBBcTAf4zvn7yZq4K4MShA.png" width="400">

#### Eslint and Typescript

We use these for our static analyisis testing. The linting and typing rules have been setup so
errors should appear in your editor if they are broken.

#### Jest

We use jest our our frontend test runner, assertion library, and spy and mock utilities for unit and
integration testing.

#### Testing-Library

We rely heavily on the different libraries that are part of the testing-library ecosystem for out
unit and integration testing. These including react-testing-library, cypress-testing-library,
react-hooks, and user-events. The guiding principles of testing-libary align with our own in that we
believe tests should resemble real world usage as closely as possible.

#### Cypress

We use cypress with cypress-testing-library as our e2e testing framework. We primarialy rely on full e2e software testing and do
not often mock API responses, but we make an exception for mocking a failed network response when
testing error states of the application.

### More Specific Examples

#### Roles and Permissions

// TODO

#### Mac and Windows Hosts

// TODO

#### Error States

// TODO

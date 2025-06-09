# Fleet UI testing

This document contains the testing strategy and plan for the frontend codebase. The testing
strategy is a high-level overview of the **who**, **what**, **when** and **why** when it comes to testing.
The testing plan primarily outlines the **how** of testing and covers the different practices and
toolings used in testing.

For instructions on using our testings tools, check out our [testing docs](https://fleetdm.com/docs/contributing/testing-and-local-development).

**Table of contents**

- [Fleet UI testing](#fleet-ui-testing)
  - [Testing strategy](#testing-strategy)
    - [Testing philosophy](#testing-philosophy)
    - [Who tests](#who-tests)
    - [What we test](#what-we-test)
      - [Shared utilities](#shared-utilities)
      - [UI building blocks (reusable components and hooks)](#ui-building-blocks-reusable-components-and-hooks)
      - [App widgets](#app-widgets)
      - [User journeys](#user-journeys)
  - [Testing plan](#testing-plan)
    - [Testing utilities for development](#testing-utilities-for-development)
    - [Types of tests](#types-of-tests)
      - [Testing cheat sheet](#testing-cheat-sheet)
      - [Manual testing](#manual-testing)
      - [Static analysis](#static-analysis)
      - [Unit testing](#unit-testing)
        - [Shared utilities testing](#shared-utilities-testing)
        - [UI building block testing](#ui-building-block-testing)
        - [Reusable hooks testing](#reusable-hooks-testing)
      - [Integration testing](#integration-testing)
        - [App widget testing](#app-widget-testing)
      - [E2E testing](#e2e-testing)
    - [Tooling](#tooling)
      - [ESLint and TypeScript](#eslint-and-typescript)
      - [Jest](#jest)
      - [Testing library](#testing-library)
    - [Additional examples](#additional-examples)
      - [Roles and permissions](#roles-and-permissions)
      - [Mac and Windows hosts](#mac-and-windows-hosts)
      - [Error states](#error-states)

---

## Testing strategy

### Testing philosophy

When we create tests, we keep in mind how an end user will be using this software. This idea
influences all other decisions when it comes to testing. Sometimes the end user is a company admin
using the Fleet UI, or a DevOps engineer using fleetctl in the terminal, or a developer using a
reusable UI component or utility or class. In any case, we should first think about the end user
when building our testing plan. Testing software from this perspective has many advantages,
including:

- A focus on functionality and behavior over implementation details. This leads to better
  maintainability of the testing suite which does not have to change as the implementation changes.
- A clear idea of what type of tests are useful and should be prioritized.
- A higher level of confidence that the software behaves as intended in real-world scenarios.

### Who tests

The **developer** is responsible for writing and maintaining tests as part of their work. They can get
help from QA when trying to decide what tests are sufficient for the feature.

### What we test

We break down our application into different types of systems that we can test in ways that best fit
their use cases.

#### Shared utilities

Shared utilities are generally simple JS functions used throughout the app. This includes code in the
`utilities` directory such as `utilites/url` and `utilities/string`.

#### UI building blocks (reusable components and hooks)

UI building blocks is reusable code that's primarily driven by props or arguments. This
includes components in the `components` directory like `Radio`, `FlashMessage`, and `Button` or
reusable hooks like `useDeepEffect`.

#### App widgets

App widgets are larger chunks of the application that are more specific (therefore not
reused) and are made up of the UI building blocks. They tend to have more context and dependencies
within them, which means they're tested differently than the building blocks. Examples include forms
like `LoginForm` and `ResetPasswordForm` or even simpler pages like `Registration Page` or `LoginPage`

#### User journeys

User journeys are the flows users take through the application to accomplish their goals. These are
typically the widest and can include navigating through multiple pages or working with multiple app
widgets on a page. This would include goals like **creating a new user or team** or **filtering
hosts by software vulnerabilities or policy results**.

---

## Testing plan

This section answers **how** we are testing our code by covering our testing tools and best practices. We
like to test the software at different layers (unit, integration, E2E), all of which have their
own usefulness and testing practices.

> NOTE: Architecture plays a huge role in testing practices and tools. In our current landscape
(__as of 08/31/2022__), we do not have the best separation of concerns between our systems. As a
result, the most reliable way to test our software is as a whole with E2E tests. While this is ok
at some level, we are still missing a large chunk of important testing that would be better written
and maintained at the integration/unit level. We'd like to utilize separation of concerns more
between our systems as this will allow us to test more in the integration and unit layers, which are
quicker to run and generally easier to work with.

### Testing utilities for development
- `make test-js` will run all frontend unit tests
- See `frontend/services/mock_service/README.md` for guidance on how to use our backend mocking infrastructure

### Types of tests

We use a variety of testing to ensure that our software is working as intended. This includes:

- End-to-end (E2E)
- Integration
- Unit
- Static
- Manual

#### Testing cheat sheet

| Systems | Type of Test | Who is the User | Example | Notes |
| ------- | ------------ | --------------- | ------- | ----- |
| Reusable utilities | Unit with Jest | Devs | String util, url util | Little to no dependencies. Function argument-based. |
| Reusable hooks | Unit with Jest & react-hooks Library | Devs | useToggleDisplayed Hook | Little to no dependencies. Function argument-based. |
| Reusable UI components | Unit with Jest & testing-library | Devs | Radio, button, and input components | Little to no dependencies. Props-based. |
| App widgets | Integration or E2E with testing-library or QA Wolf | End users | Create user form. Reset password form. | Less reusable code with more complex environment setup and dependencies. Depending on the case, can be done with integration or E2E. For integration, mock at the backend level; don't mock other UI systems. For E2E, don't mock other systems except for common network error states. |
| User journeys | E2E with QA Wolf | End users | Filtering a host by software. Creating a team as admin. | Full business flows. Little to no mocking of systems, except for common network error states. |
| N/A | Manual | N/A | // TODO | Manual testing can be used for all types of code. Examples would be for one-offs or states that would require extremely difficult testing setups. |

#### Manual testing

There will always be a space for manual testing. We use it for testing states that do not
occur very often in the application or are not worth the effort to test for unlikely edge cases.

#### Static analysis

This includes typing and linting to quickly ensure proper typings of data flowing through the
application, and that we are following coding conventions and styling rules. This gives us a first
line of defense against writing buggy code.

#### Unit testing

We unit test smaller reusable components that have little to no dependencies within them. These
components are primarily parametric-based and require no or minimal mocking to be tested
effectively. They tend to be small building blocks of our application (e.g., reusable UI components,
common utilities, reusable hooks). With unit testing, the end users tend to be other developers, so
we ensure these components work as expected when used as building blocks.

##### Shared utilities testing

We can test utility functions purely with Jest and don't have to worry about rendering components with
react-testing-library. Only Jest is needed with minimal mocking.

[View the full url utility testing source](https://github.com/fleetdm/fleet/blob/main/frontend/utilities/url/url.tests.ts).

```tsx
import { buildQueryStringFromParams } from ".";

describe("url utilities", () => {
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
##### UI building block testing

We test the component to ensure it works how a developer would want to use it. There is minimal
mocking and we utilize react-testing-library and Jest spies to test.

[View the full Radio component testing source](https://github.com/fleetdm/fleet/blob/main/frontend/components/forms/fields/Radio/Radio.tests.tsx).

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

##### Reusable hooks testing

// TODO

#### Integration testing

We use integration testing to strike a balance between speed and expense to write our tests. We use
them to test multiple reusable components that come together into less reusable app widgets. We also
try to use minimal mocking, but we can mock at the backend level if required.

##### App widget testing

This layer of tests is great for testing difficult to obtain states and edge cases as the test setup
is simpler than E2E tests. We highly utilize react-testing-library to interface with these components.

[View the full ResetPasswordForm app widget testing source](https://github.com/fleetdm/fleet/blob/main/frontend/components/forms/ResetPasswordForm/ResetPasswordForm.tests.jsx).

```tsx
import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

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

Our E2E layer tests all the systems of the software (frontend and backend) together to
ensure the application works as intended. We have partnered with QA Wolf to cover these flows, and
the E2E tests are written and maintained by them.

The code is deployed and tested once daily on the testing instance. However, development may necessitate running E2E tests on demand. To run E2E tests live on a branch such as the `main` branch, developers can navigate to [Deploy Cloud Environments](https://github.com/fleetdm/confidential/actions/workflows/cloud-deploy.yml) in our [/confidential](https://github.com/fleetdm/confidential) repo's Actions and select "Run workflow".


### Tooling

Here is a quick reference of the current tooling we are using at each layer of testing.

<img src="https://miro.medium.com/max/1400/1*iBBcTAf4zvn7yZq4K4MShA.png" width="400">

#### ESLint and TypeScript

We use these for our static analysis testing. These tools have been set up so
errors should appear in your editor if they are broken.

#### Jest

We use Jest as our frontend test runner, assertion library, and spy and mock utilities for unit and
integration testing.

#### Testing library

We rely heavily on the different libraries that are part of the testing-library ecosystem for our
unit and integration testing. These including react-testing-library, react-hooks, and user-events.
The guiding principles of the testing-library tools align with our own
in that we believe tests should resemble real-world usage as closely as possible.

### Additional examples

#### Roles and permissions

// TODO

#### Mac and Windows hosts

// TODO

#### Error states

// TODO

<meta name="pageOrderInSection" value="250">
<meta name="description" value="Learn about the testing strategy and plan for Fleet's frontend codebase.">

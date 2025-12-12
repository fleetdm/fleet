# Fleet frontend

The Fleet frontend is a Single Page Application using React with Typescript and Hooks.

## Table of contents
- [Running the Fleet web app](#running-the-fleet-web-app)
- [Testing](#testing)
- [Directory Structure](#directory-structure)
- [Patterns](#patterns)
- [Storybook](#storybook)

## Running the Fleet web app

For details instruction on building and serving the Fleet web application
consult the [Contributing documentation](../docs/Contributing/getting-started/README.md).

## Testing

Visit the [overview of Fleet UI testing](../docs/Contributing/guides/ui/fleet-ui-testing.md) for more information on our testing strategy, philosophies, and tools.

To run unit or integration tests in `ComponentName.tests.tsx`, run `yarn test -- ComponentName.tests.tsx`. To [test all Javascript components](https://fleetdm.com/docs/contributing/testing-and-local-development#javascript-unit-tests) run `yarn test`.


[QA Wolf](https://www.qawolf.com/) manages our E2E test and will maintain the tests as well as raise
any issues found from these tests. Engineers should not have to worry about working with E2E testing
code or raising issues themselves.

For more information on how our front-end tests work, visit our [frontend test
directory](./test/README.md).

## Directory structure

Component directories in the Fleet front-end application encapsulate the entire
component, including files for the component and its styles. The
typical directory structure for a component is as follows:

```
└── ComponentName
  ├── _styles.scss
  ├── ComponentName.tsx
  |-- ComponentName.tests.tsx
  ├── index.ts
```

- `_styles.scss`: The component css styles
- `ComponentName.tsx`: The React component
- `ComponentName.tests.tsx`: The React component unit/integration tests
- `index.ts`: Exports the React component
  - This file is helpful as it allows other components to import the component
    by it's directory name. Without this file the component name would have to
    be duplicated during imports (`components/ComponentName` vs. `components/ComponentName/ComponentName`).

### [components](./components)

The component directory contains global React components rendered by pages, receiving props from
their parent components to render data and handle user interactions.

### [context](./context)

The context directory contains the React Context API pattern for various entities.
Only entities that are needed across the app has a global context. For example,
the [logged in user](./context/app.tsx) (`currentUser`) has multiple pages and components
where its information is pulled.

### [interfaces](./interfaces)

Files in the interfaces directory are used to specify the Typescript interface for a reusable Fleet
entity. This is designed to DRY up the code and increase re-usability. These
interfaces are imported in to component files and implemented when defining the
component's props.

**Additionally, local interfaces are used for props of local components.**

### [layouts](https://github.com/fleetdm/fleet/tree/main/frontend/layouts)

The Fleet application has only 1 layout, the [Core Layout](./layouts/CoreLayout/CoreLayout.jsx).
The Layout is rendered from the [router](./router/index.tsx) and are used to set up the general
app UI (header, sidebar) and render child components.
The child components rendered by the layout are typically page components.

### [pages](./pages)

Page components are React components typically rendered from the [router](./router).
React Router passed props to these pages in case they are needed. Examples include
the `router`, `location`, and `params` objects.

### [router](./router)

The router directory is where the react router lives. The router decides which
component will render at a given URL. Components rendered from the router are
typically located in the [pages directory](./pages). The router directory also holds a `paths`
file which holds the application paths as string constants for reference
throughout the app. These paths are typically referenced from the [App
Constants](./app_constants) object.

### [services](./services)

CRUD functions for all Fleet entities (e.g. `query`) that link directly to the Fleet API.

### [styles](./styles)

The styles directory contains the general app style setup and variables. It
includes variables for the app color hex codes, fonts (families, weights and sizes), and padding.

### [templates](./templates)

The templates directory contains the HTML file that renders the React application via including the `bundle.js`
and `bundle.css` files. The HTML page also includes the HTML element in which the React application is mounted.

### [test](./test)

The test directory includes test helpers, API request mocks, and stubbed data entities for use in test files.
See [the UI testing documentation](./test/README.md) for more on test helpers, stubs, and request mocks.

### [utilities](./utilities)

The utilities directory contains re-usable functions and constants for use throughout the
application. The functions include helpers to convert an array of objects to
CSV, debounce functions to prevent multiple form submissions, format API errors,
etc.

## Patterns

The list of patterns used in the Fleet UI codebase can be found [in `patterns.md`](./docs/patterns.md).

## Storybook

[Storybook](https://storybook.js.org/) is a tool to document and visualize components, and we
use it to capture our global components used across Fleet. Storybook is key when developing new
features and testing components before release. It runs a separate server exposed on port `6006`.
To run this server, do the following:

- Go to your root fleet project directory
- Run `make deps`
- Run `yarn storybook`

The URL `localhost:6006` should automatically show in your browser. If not, visit it manually.

Running Storybook before implementing new UI elements can clarify if new components need to be created or already exist. When creating a component, you can create a new file, `component.stories.tsx`, within its directory. Then, fill it with the appropriate Storybook code to create a new Storybook entry. You will be able to visualize the component within Storybook to determine if it looks and behaves as expected.

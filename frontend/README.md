# Fleet Front-End

The Fleet front-end is a Single Page Application using React with Typescript and Hooks.

## Table of Contents
- [Running the Fleet web app](#running-the-fleet-web-app)
- [Directory Structure](#directory-structure)
- [Deprecated](#deprecated)
- [Patterns](#patterns)
  - [Typing](#typing)
  - [React Hooks (Functional Components)](#react-hooks-functional-components)
  - [React Context](#react-context)
  - [Fleet API Calls](#fleet-api-calls)
  - [Page Routing](#page-routing)
  - [Other](#other)

## Running the Fleet web app

For details instruction on building and serving the Fleet web application
consult the [Contributing documentation](../docs/3-Contributing/README.md).

## Directory Structure

Component directories in the Fleet front-end application encapsulate the entire
component, including files for the component and its styles. The
typical directory structure for a component is as follows:

```
|-- ComponentName
|  |-- _styles.scss
|  |-- ComponentName.tsx
|  |-- index.ts
```

- `_styles.scss`: The component css styles
- `ComponentName.tsx`: The React component
- `index.ts`: Exports the React component
  - This file is helpful as it allows other components to import the component
    by it's directory name. Without this file the component name would have to
    be duplicated during imports (`components/ComponentName` vs. `components/ComponentName/ComponentName`).

### [app_constants](./app_constants)

The app_constants directory exports the constants used in the app. Examples
include the app's URL paths, settings, and http statuses. When building features
that require constants, the constants should be added here for accessibility
throughout the application.

### [components](./components)

The component directory contains the React components rendered by pages. They
are typically not connected to the redux state but receive props from their
parent components to render data and handle user interactions.

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
The Layout is rendered from the [router](./router/index.jsx) and are used to set up the general 
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

### [utilities](./utilities)

The utilities directory contains re-usable functions and constants for use throughout the
application. The functions include helpers to convert an array of objects to
CSV, debounce functions to prevent multiple form submissions, format API errors,
etc.

## Deprecated

These directories and files are still used (as of 9/14/21) but are being replaced by newer code:

- [fleet](./fleet), now using [services](./services)
- [redux](./redux), now using [services](./services), local states, and various entities directly (e.g. React Router)
- [Form.jsx Higher Order Component](./components/forms/README.md), now creating forms with local states with React Hooks (i.e. `useState`)

## Patterns

### Typing
All Javascript and React files are using Typescript, which means the extensions used are `.ts` and `.tsx`.
The amount of typing across the app is a preference by team. Here are the guidelines on what we type at Fleet:

- Page/Component props (using an interface)
  - Destructure props in page/component signature

  ```js
  // page
  interface IPageProps {
    prop1: string;
    prop2: number;
    ...
  }

  const PageOrComponent = ({
    prop1,
    prop2,
  }: IPageProps) => {
    
    return (
      // ...
    );
  };
  ```

- Local states 
```js 
const [item, setItem] = useState<string>("");
```

- Fetch function signature (i.e. `react-query`)
```js
useQuery<IHostResponse, Error, IHost>(params)
```

- Parameters of a function (including callbacks)
```js
const functionWithTableName = (tableName: string) => {
  // do something
};
```

**Note: Local interfaces are used as well as global entity interfaces.**

### React Hooks (Functional Components)

[Hooks](https://reactjs.org/docs/hooks-intro.html) are used to track state and use other features
of React. Hooks are only allowed in functional components, which are created like so:

```js
import React, { useState, useEffect } from "React";

const PageOrComponent = (props) => {
  const [item, setItem] = useState<string>("");

  // runs only on first mount (replaces componentDidMount)
  useEffect(() => {

  }, []);

  // runs only when `item` changes (replaces componentDidUpdate) 
  useEffect(() => {

  }, [item]);
  
  return (
    // ...
  );
};
```

### React Context

[React Context](https://reactjs.org/docs/context.html) is a store similar to Redux. It stores 
data that is desired and allows for retrieval of that data in whatever component is in need.
View currently working contexts in the [context directory](./context).

### Fleet API Calls

**Background:** 

Redux was used to make API calls, along with the [fleet](./fleet) directory.

**Current:**

The [services](./services) directory stores all API calls and is to be used in two ways: 
- A direct `async/await` assignment
- Using `react-query` if requirements call for loading data right away or based on dependencies. 

Examples below:

*Direct assignment*
```js
// page
import ...
import queryAPI from "services/entities/queries";

const PageOrComponent = (props) => {
  const doSomething = async () => {
    const response = await queryAPI.load(param);
    // do something
  };
  
  return (
    // ...
  );
};
```

*React Query*

`react-query` ([docs here](https://react-query.tanstack.com/overview)) is a data-fetching library that
gives us the ability to fetch, cache, sync and update data through a myriad of options and properties.

```js
import ...
import { useQuery, useMutation } from "react-query";
import queryAPI from "services/entities/queries";

const PageOrComponent = (props) => {
  // retrieve the query based on page/component load
  // and dependencies for when to refetch
  const {
    isLoading,
    data,
    error,
    ...otherProps
  } = useQuery<IResponse, Error, IData>(
    "query",
    () => queryAPI.load(param),
    {
      ...options
    }
  );

  // `props` is a bucket of properties that can be used when 
  // updating data. for example, if you need to know whether
  // a mutation is loading, there is a prop for that
  const { mutateAsync } = useMutation((formData: IForm) =>
    queryAPI.create(formData)
  );
  
  return (
    // ...
  );
};
```

### Page Routing
In the past, we used Redux to manage redirecting to different pages
of the app. Now, we are using React Router directly. For all pages,
React Router (v3) supplies a `router` prop that can be easily accessed.
When needed, the `router` object contains a `push` function that redirects
a user to whatever page desired. For example:

```js
// page
import PATHS from "router/paths";

interface IPageProps {
  router: any; // no typing in react-router v3
}

const PageOrComponent = ({
  router,
}: IPageProps) => {
  const doSomething = async () => {
    router.push(PATHS.HOME);
  };
  
  return (
    // ...
  );
};
```

### Other

**Local states**
Our first line of defense for state management is local states (i.e. `useState`). We
use local states to keep pages/components separate from one another and easy to 
maintain. If states need to be passed to direct children, then prop-drilling should 
suffice as long as we do not go more than two levels deep. Otherwise, if states need 
to be used across multiple unrelated components or 3+ levels from a parent, 
then the [app's context](#react-context) should be used. 

**File size**
The recommend line limit per page/component is 500 lines. This is only a recommendation.
Larger files are to be split into multiple files if possible.
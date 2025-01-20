# Patterns

This contains the patterns that we follow in the Fleet UI.

> NOTE: There are always exceptions to the rules, but we try as much as possible to
follow these patterns unless a specific use case calls for something else. These
should be discussed within the team and documented before merged.

## Table of contents

- [Typing](#typing)
- [Utilities](#utilities)
- [Components](#components)
- [Forms](#forms)
- [React hooks](#react-hooks)
- [React Context](#react-context)
- [Fleet API calls](#fleet-api-calls)
- [Page routing](#page-routing)
- [Styles](#styles)
- [Icons and images](#icons-and-images)
- [Testing](#testing)
- [Security considerations](#security-considerations)
- [Other](#other)

## Typing

All Javascript and React files use Typescript, meaning the extensions are `.ts` and `.tsx`. Here are the guidelines on how we type at Fleet:

- Use *[global entity interfaces](../README.md#interfaces)* when interfaces are used multiple times across the app
- Use *local interfaces* when typing entities limited to the specific page or component

### Local interfaces for page, widget, or component props

```typescript
// page
interface IPageProps {
  prop1: string;
  prop2: number;
  ...
}

// Note: Destructure props in page/component signature
const PageOrComponent = ({ prop1, prop2 }: IPageProps) => {
  // ...
};
```

### Local states with types

```typescript
// Use type inference when possible.
const [item, setItem] = useState("");

// Define the type in the useState generic when needed.
const [user, setUser] = useState<IUser>()
```

### Fetch function signatures (i.e. `react-query`)

```typescript
// include the types for the response, error.
const { data } = useQuery<IHostResponse, Error>(
  'host',
  () => hostAPI.getHost()
)


// include the third host data generic argument if the response data and exposed data are different.
// This is usually the case when we use the `select` option in useQuery.

// `data` here will be type IHostProfiles
const { data } = useQuery<IHostResponse, Error, IHostProfiles>(
  'host',
  () => hostAPI.getHost()
  {
    // `data` here will be of type IHostResponse
    select: (data) => data.profiles
  }
)
```

### Functions

```typescript
// Type all function arguments. Use type inference for the return value type.
// NOTE: sometimes typescript does not get the return argument correct, in which
// case it is ok to define the return type explicitly.
const functionWithTableName = (tableName: string)=> {
  // ...
};
```

### API interfaces

```typescript
// API interfaces should live in the relevant entities file.
// Their names should be named to clarify what they are used for when interacting
// with the API

// should be defined in service/entities/hosts.ts
interface IHostDetailsReponse {
  ...
}
interface IGetHostsQueryParams {
  ...
}

// should be defined in service/entities/teams.ts
interface ICreateTeamPostBody {
  ...
}

```

## Utilities

### Named exports

We export individual utility functions and avoid exporting default objects when exporting utilities.

```ts

// good
export const replaceNewLines = () => {...}

// bad
export default {
  replaceNewLines
}
```

## Components

### React functional components

We use functional components with React instead of class comonents. We do this
as this allows us to use hooks to better share common logic between components.

### Passing props into components
We tend to use explicit assignment of prop values, instead of object spread syntax:
```
<ExampleComponent prop1={pop1Val} prop2={prop2Val} prop3={prop3Val} />
```

### Naming handlers
When defining component props for handlers, we prefer naming with a more general `onAction`. When
naming the handler passed into that prop or used in the same component it's defined, we prefer
either the same `onAction` or, if useful, a more specific `onMoreSpecifiedAction`. E.g.:

```tsx
<BigSecretComponent
  onSubmit={onSubmit} 
/>
```
or
```tsx
<BigSecretComponent
  onSubmit={onUpdateBigSecret}
/>
```

### Page component pattern

When creating a **top level page** (e.g. dashboard page, hosts page, policies page)
we wrap that page's content inside components `MainContent` and
`SidePanelContent` if a sidebar is needed.

These components encapsulate the styling used for laying out content and also
handle rendering of common UI shared across all pages (current this is only the
sandbox expiry message with more to come).

```typescript
/** An example of a top level page utilising MainConent and SidePanel content */
const PackComposerPage = ({ router }: IPackComposerPageProps): JSX.Element => {
  // ...

  return (
    <>
      <MainContent className={baseClass}>
        <PackForm
          className={`${baseClass}__pack-form`}
          handleSubmit={handleSubmit}
          onFetchTargets={onFetchTargets}
          selectedTargetsCount={selectedTargetsCount}
          isPremiumTier={isPremiumTier}
        />
      </MainContent>
      <SidePanelContent>
        <PackInfoSidePanel />
      </SidePanelContent>
    </>
  );
};

export default PackComposerPage;
```


## Forms

### Form buttons

When building a React-controlled form, use the `Button` component for its submit button, and do not
include a `type`, allowing it to default to `type="button"`, which is what we want. Using
`type="submit"` leads to confusing behavior wherein the native HTML submit functionality
interferes with our custom submit logic, causing the request to be canceled in a messy and hard to
recognize way. Alternatively, if using `type="submit"`, ensure the submit handler calls `evt.preventDefault`

### Data validation

#### How to validate
Forms should make use of a pure `validate` function whose input(s) correspond to form data (may include
new and possibly former form data) and whose output is an object of formFieldName:errorMessage
key-value pairs (`Record<string,string>`) e.g.

```
const validate = (newFormData: IFormData) => {
  const errors = {};
  ...
  return errors;
}
```
The output of `validate` should be used by the calling handler to set a `formErrors`
state.

#### When to validate
Form fields should *set only new errors* on blur and on save, and *set or remove* errors on change. This provides
an "optimistic" user experience. The user is only told they have an error once they navigate
away from a field or hit enter, actions which imply they are finished editing the field, while they are informed they have fixed
an error as soon as possible, that is, as soon as they make the fixing change. e.g.
```
const onInputChange = ({ name, value }: IFormField) => {
  const newFormData = { ...formData, [name]: value };
  setFormData(newFormData);
  const newErrs = validateFormData(newFormData);
  // only set errors that are updates of existing errors
  // new errors are only set onBlur
  const errsToSet: Record<string, string> = {};
  Object.keys(formErrors).forEach((k) => {
    if (newErrs[k]) {
      errsToSet[k] = newErrs[k];
    }
  });
  setFormErrors(errsToSet);
};

```

,

```
const onInputBlur = () => {
  setFormErrors(validateFormData(formData));
};
```

, and 

```
const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
  evt.preventDefault();

  // return null if there are errors
  const errs = validateFormData(formData);
  if (Object.keys(errs).length > 0) {
    setFormErrors(errs);
    return;
  }

  ...
  // continue with submit logic if no errors

```

## React hooks

[Hooks](https://reactjs.org/docs/hooks-intro.html) are used to track state and use other features
of React. Hooks are only allowed in functional components, which are created like so:
  
```typescript
import React, { useState, useEffect } from "React";

const PageOrComponent = (props) => {
  const [item, setItem] = useState("");

  // runs only on first mount (replaces componentDidMount)
  useEffect(() => {
    // do something
  }, []);

  // runs only when `item` changes (replaces componentDidUpdate)
  useEffect(() => {
    // do something
  }, [item]);

  return (
    // ...
  );
};
```

> NOTE: Other hooks are available per [React's documentation](https://reactjs.org/docs/hooks-intro.html).

## React context

[React context](https://reactjs.org/docs/context.html) is a state management store. It stores
data that is desired and allows for retrieval of that data in whatever component is in need.
View currently working contexts in the [context directory](../context).

## Fleet API calls

The [services](../services) directory stores all API calls and is to be used in two ways:
- A direct `async/await` assignment
- Using `react-query` if requirements call for loading data right away or based on dependencies.

Examples below:

**Direct assignment**

```typescript
// page
import ...
import queriesAPI from "services/entities/queries";

const PageOrComponent = (props) => {
  const doSomething = async () => {
    try {
      const response = await queriesAPI.load(param);
      // do something
    } catch(error) {
      console.error(error);
      // maybe trigger renderFlash
    }
  };

  return (
    // ...
  );
};
```

**React Query**

[react-query](https://react-query.tanstack.com/overview) is a data-fetching library that
gives us the ability to fetch, cache, sync and update data with a myriad of options and properties.

```typescript
import ...
import { useQuery, useMutation } from "react-query";
import queriesAPI from "services/entities/queries";

const PageOrComponent = (props) => {
  // retrieve the query based on page/component load
  // and dependencies for when to refetch
  const {
    isLoading,
    data,
    error,
    ...otherProps,
  } = useQuery<IResponse, Error, IData>(
    "query",
    () => queriesAPI.load(param),
    {
      ...options
    }
  );

  // `props` is a bucket of properties that can be used when
  // updating data. for example, if you need to know whether
  // a mutation is loading, there is a prop for that.
  const { ...props } = useMutation((formData: IForm) =>
    queriesAPI.create(formData)
  );

  return (
    // ...
  );
};
```

## Page routing

We use React Router directly to navigate between pages. For page components,
React Router (v3) supplies a `router` prop that can be easily accessed.
When needed, the `router` object contains a `push` function that redirects
a user to whatever page desired. For example:

```typescript
// page
import PATHS from "router/paths";
import { InjectedRouter } from "react-router/lib/Router";

interface IPageProps {
  router: InjectedRouter; // v3
}

const PageOrComponent = ({
  router,
}: IPageProps) => {
  const doSomething = () => {
    router.push(PATHS.DASHBOARD);
  };

  return (
    // ...
  );
};
```

## Styles

Below are a few need-to-knows about what's available in Fleet's CSS:



### Modals

1) When creating a modal with a form inside, the action buttons (cancel, save, delete, etc.) should
   be wrapped in the `modal-cta-wrap` class to keep unified styles.

### Forms

1) When creating a form, **not** in a modal, use the class `${baseClass}__button-wrap` for the
   action buttons (cancel, save, delete, etc.) and proceed to style as needed.


## Icons and images

### Adding icons

To add a new icon:

1. create a React component for the icon in `frontend/components/icons` directory. We will add the
   SVG here.
2. download the icon source from Figma as an SVG file
3. run the downloaded file through an SVG optimizer such as
   [SVGOMG](https://jakearchibald.github.io/svgomg/) or [SVG Optimizer](https://svgoptimizer.com/)
4. download the optimized SVG and place it in created file from step 1.
5. import the new icon in the `frontend/components/icons/index.ts` and add it the the `ICON_MAP`
   object. The key will be the name the icon is accessible under.

The icon should now be available to use with the `Icon` component from the given key name.

```tsx
// using a new icon with the given key name 'chevron`
<Icon name="chevron" />
```



### File size

The recommend line limit per page/component is 500 lines. This is only a recommendation.
Larger files are to be split into multiple files if possible.


## Testing

At a bare minimum, we make every effort to test that components that should render data are doing so
as expected. For example: `HQRTable.tests.tsx` tests that the `HQRTable` component correctly renders
data being passed to it.

At a bare minimum, critical bugs released involving the UI will have automated testing discussed at the critical bug post-mortem with a frontend engineer and an engineering manager. We make every effort to add an automated test to either the unit, integration, or E2E layer to prevent the critical bug from resurfacing.

## Security considerations

We make every effort to avoid using the `dangerouslySetInnerHTML` prop. When absolutely necessary to
use this prop, we make sure to sanitize any user-defined input to it with `DOMPurify.sanitize`

## Other

### Local states

Our first line of defense for state management is local states (i.e. `useState`). We
use local states to keep pages/components separate from one another and easy to
maintain. If states need to be passed to direct children, then prop-drilling should
suffice as long as we do not go more than two levels deep. Otherwise, if states need
to be used across multiple unrelated components or 3+ levels from a parent,
then the [app's context](#react-context) should be used.

### Reading and updating configs

If you are dealing with a page that *updates* any kind of config, you'll want to access that config
with a fresh API call to be sure you have the updated values. Otherwise, that is, you are dealing
with a page that is only *reading* config values, get them from context.

### Rendering flash messages

Flash messages by default will be hidden when the user performs any navigation that changes the URL,
in addition to the timeout set for success messages. The `renderFlash` method from notification
context accepts an optional third `options` argument which contains an optional
`persistOnPageChange` boolean field that can be set to `true` to negate this default behavior.

If the `renderFlash` is accompanied by a router push, it's important to push to the router *before*
calling `renderFlash`. If the push comes after the `renderFlash` call,
the flash message may register the `push` and immediately hide itself.

```tsx
// first push
router.push(newPath);
// then flash
renderFlash("error", "Something went wrong");
```
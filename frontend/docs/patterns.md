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
- [Tier modes](#tier-modes)
- [React hooks](#react-hooks)
- [React Context](#react-context)
- [Fleet API calls](#fleet-api-calls)
- [Page routing](#page-routing)
- [Command palette](#command-palette)
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
// Their names should clarify what they are used for when interacting with the
// API. In service functions, prefer `formData` as the variable name for request
// bodies to stay consistent with the *FormData interface naming convention.

// should be defined in service/entities/hosts.ts
interface IHostDetailsResponse {
  ...
}
interface IGetHostsQueryParams {
  ...
}

// should be defined in service/entities/users.ts
interface IUpdateUserFormData {
  ...
}

// should be defined in service/entities/software.ts
interface IGetSoftwareApiParams {
  ...
}
interface ISoftwareCountResponse {
  ...
}

// Use *FormData for form-driven bodies, *ApiParams/*QueryParams for request
// params, *Response for responses, *QueryKey when typing a React Query key.
// Avoid *Body, *PostBody, *Payload, *Request for API request bodies — use
// *FormData instead, even for programmatic request bodies (e.g.
// IDeleteQueriesFormData). One consistent suffix is easier to follow than
// asking each dev to judge "is this form-driven enough?"
// *PreviewPayload is fine for outgoing webhook shapes.
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

### Software titles

Software titles have two fields that look like a name:

- `name` — the raw title from the installer/package metadata (e.g. `Microsoft.CompanyPortal`)
- `display_name` — an optional custom name set per fleet by an admin

Render the label from the resolved display name, but pass the **raw** `name` to
`<SoftwareIcon>` — the icon matcher only knows raw, well-known names.

#### Display name

**Never render `name` directly in the UI.** Always route software names through
`getDisplayedSoftwareName(name, display_name)` from `pages/SoftwarePage/helpers.tsx`.
It prefers `display_name`, normalizes known awkward titles (e.g.
`microsoft.companyportal` → `Company Portal`), and falls back to a sensible
default. This applies everywhere a software title is shown: table rows, dropdown
options, modal text, activity feed entries, automation summaries, etc.

```tsx
// good
label: getDisplayedSoftwareName(title.name, title.display_name),

// bad — misses display_name and the WELL_KNOWN_SOFTWARE_TITLES normalization
label: title.name,

// also bad — misses the WELL_KNOWN_SOFTWARE_TITLES normalization
label: title.display_name || title.name,
```

The same rule applies to any object shape that carries both fields
(`ISoftwareTitle`, `ISoftwarePackage`, `IAppStoreApp`, `IHostSoftware`,
`IPolicySoftwareToInstall`, etc.). The `ISoftwareTitle.name` JSDoc states the
expectation: "All software names displayed by UI is ran through
getDisplayedSoftwareName."

#### Icons

`<SoftwareIcon name={...}>` uses the `name` prop for **fallback icon matching**
via `getMatchedSoftwareIcon({ name, source })` when `icon_url` is null. That
matcher only knows the raw, well-known names (`notion`, `microsoft.companyportal`,
etc.). If you pass it a resolved display name like `getDisplayedSoftwareName(...)`
or `display_name || name`, an admin who renames the title to anything not in the
match table will lose the icon to a generic fallback. Fleet-maintained apps are
the highest-risk surface because they have no `icon_url` — they depend
entirely on name matching. See #47123.

When you have both fields, pass the raw `name` to the icon and the resolved
name to the label:

```tsx
// good — icon matches against raw name, label shows resolved display name
const displayName = getDisplayedSoftwareName(title.name, title.display_name);
<>
  <SoftwareIcon name={title.name} source={title.source} url={title.icon_url} />
  {displayName}
</>

// bad — admin renames break the icon match for FMAs and other matched titles
<SoftwareIcon name={displayName} ... />
```

When the data has been flattened into a single `name` field upstream (e.g. for a
row object or table-cell renderer), carry the raw name alongside it as a
separate field (`iconName`, `rawName`, etc.) and feed THAT to `<SoftwareIcon>`.
See `frontend/pages/policies/ManagePoliciesPage/helpers.tsx`'s
`ISoftwareAutomationData.iconName` for the established pattern.

`SoftwareNameCell` already does this internally — when you can use it, prefer
it over hand-rolling icon + label rendering.

## Components

### React functional components

We use functional components with React instead of class comonents. We do this
as this allows us to use hooks to better share common logic between components.

### Passing props into components

We strongly prefer explicit assignment of prop values over object spread syntax. In almost all cases, list every prop by name:

```tsx
<ExampleComponent prop1={prop1Val} prop2={prop2Val} prop3={prop3Val} />
```

Spreading is hard to review (the reader can't see what's being passed), brittle under refactors (adding a key to the source bag silently changes the target), and on native DOM elements it's a real security footgun — anything in the bag (including `dangerouslySetInnerHTML`, `href`, `src`, event handlers) gets applied.

#### Accepted exceptions

Spread is acceptable in these cases:

- **react-select 5 custom subcomponents** — the library contract requires forwarding the full internal props bag (`innerRef`, `innerProps`, `selectProps`, …) to `components.X`. The bag is library-generated, not user input.
- **react-table v7 prop getters** (`getCellProps()`, `getRowProps()`, `getHeaderProps()`, `getToggleAllRowsSelectedProps()`) — the prop-getter pattern *is* the library's API. Cell data is rendered through `cell.render("Cell")`, never as attributes.
- **react-markdown renderer overrides** (e.g. `code: ({...props}) => <code {...props}>`) — the bag is library-controlled HAST metadata, not raw markdown. Do not enable the `rehype-raw` plugin: it lets raw HTML from the markdown source pass through to the bag, and the spread would forward it straight to the DOM — author-controlled HTML rendering verbatim is an XSS sink.
- **Typed SVG icon components** (`SVGProps<SVGSVGElement>` flowing into `<svg>`, as in `pages/SoftwarePage/components/icons/*`) — the `SVGProps` type constrains callers to valid SVG attributes. Do *not* widen the prop type to `any` or `Record<string, unknown>`; that removes the guard that makes this safe.
- **Test helpers, factories, and Storybook stories** — non-production code. The bag is built locally in the same file by code that owns its shape.

#### Not safe — never spread

Never spread props (especially anything derived from API responses, URLs, markdown source, MDM payloads, host facts, software metadata, or other external data) onto:

`<a>`, `<img>`, `<iframe>`, `<object>`, `<embed>`, `<source>`, `<link>`, `<script>`, `<form>`, `<video>`, `<audio>`.

For those elements, pick out `href` / `src` / etc. explicitly and validate the value (scheme allowlist, no `javascript:` URIs, etc.) before passing it.

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
    <SidePanelPage>
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
  </SidePanelPage>
  );
};

export default PackComposerPage;
```

## Forms

### Form submission

When building a React-controlled form:
- Use the native HTML `form` element to wrap the form.
- Use a `Button` component with `type="submit"` for its submit button.
- Write a submit handler, e.g. `handleSubmit`, that accepts an `evt: React.FormEvent<HTMLFormElement>` argument and, critically:
  - calls `evt.preventDefault()` in its body. This prevents the HTML `form`'s default submit behavior from interfering with our custom handler's logic.
  - runs `validate` against the full form, sets errors on every invalid field, and returns without submitting when any errors are present.
- Assign that handler to the `form`'s `onSubmit` property (*not* the submit button's `onClick`).
- Disable the submit button only while a submission is in flight. Do not disable it because required fields are empty or values are currently invalid — see [Submit button state](#submit-button-state).

### Data validation

The rules below describe the target behavior. Existing forms implement the rules directly and are migrated one at a time. New forms should follow these rules on day one.

#### How to validate

Forms use a pure `validate` function whose input is the current form data and whose output is a `Record<string, string>` of `fieldName → errorMessage` pairs. Only invalid fields appear in the output.

```tsx
const validate = (formData: IFormData): Record<string, string> => {
  const errors: Record<string, string> = {};
  if (!formData.email.trim()) {
    errors.email = "Enter your email";
  } else if (!isValidEmail(formData.email)) {
    errors.email = "Enter a valid email";
  }
  return errors;
};
```

The output of `validate` is used by the calling handler to update a `formErrors` state that feeds each `InputField`'s `error` prop.

#### When errors appear

- Never show a field's error before the user has interacted with that field. A field becomes interacted-with when the user types into it or blurs it while it holds a value.
- On blur of a field the user has interacted with, run validation and show the resulting error (if any) for that field only. Do not touch errors on other fields.
- On submit, show inline errors on every invalid field simultaneously, then return without submitting.
- Autofill counts as user interaction — treat autofilled fields as touched.
- On an Edit form, pre-filled values that are invalid do not show errors until the user interacts with the field.

#### When errors clear

- On focus (click-in) of a field that has an error, clear that field's error immediately — do not wait for the user to type a valid value. The error text replaces the field's label (see [Visual affordances](#visual-affordances)), so clearing on focus restores the label and lets the user see what they're editing.
- Re-validate on blur, not on keystroke.
- Typing in one field never clears errors on other fields. Clearing is per-field.
- When a validation becomes irrelevant (e.g. a conditional requirement is removed by toggling a checkbox), clear the newly-irrelevant error immediately.

#### Error priority

- Presence errors take priority over format errors. If a field is both empty and format-invalid, show the presence error.
- Show one error per field at a time. Never stack multiple errors on the same field.
- Server-set errors follow the same "one at a time" rule.

#### Submit button state

- The submit button is enabled by default. Empty required fields, currently-invalid values, unchanged Edit forms, and prior server errors do not disable it.
- The only reasons to disable the submit button are an in-flight submission (see [In-flight and submission lifecycle](#in-flight-and-submission-lifecycle)) or the entire form being disabled by GitOps mode. GitOps-managed pages disable the form's fields and submit button together — users cannot save through the UI at all.
- If the user clicks submit with invalid data, the submit handler shows all inline errors and returns without submitting. The button itself stays enabled so the click can surface the errors.
- Do not gate on `isDirty` or "form has changes." A no-op re-save is allowed.

#### Server-side errors

- Field-specific server errors (e.g. "email already taken") render inline on the field via the same `error` prop as client-side errors, AND fire a toast with the error message. Submit stays enabled. The toast is intentional even when the inline error is visible — forms can be long enough that the errored field is scrolled off-screen after submission.
- Cross-field or global server errors (e.g. `formatErrorResponse` `.base`) surface as a toast only. No inline surface.
- When the user focuses a field that has a server-set error, clear it immediately — same rule as client-side.
- Multiple field errors returned by the server: iterate the error map and set all inline. Prefer a single summary toast when there are many.

#### Conditional / dependent validation

- Cross-field checks (e.g. password + confirmation match) run on blur of either field. The error attaches to the field that is invalid, not to both.
- Fields that become required based on another field's state (e.g. password required when SSO is off) still follow the "no error until interacted with" rule. There is no visual indicator that a field is conditionally required.
- When a condition changes such that an existing error no longer applies (e.g. SSO toggled on), clear the error immediately.
- Client-side "at least one X must be selected" errors render inline on the selector's label, not as a toast. Server-side variants of the same error also fire a toast in addition to the inline surface.

#### Optional and disabled fields

- Empty optional fields never show an error.
- An optional field that has a value with a format constraint (e.g. an optional email field) validates the format. On invalid, show an inline error, but do not disable submit.
- Disabled fields skip validation entirely. A disabled field is never in an error state, regardless of its value.

#### Input hygiene

- Trim leading and trailing whitespace client-side before submitting. Send the trimmed value to the API.
- Whitespace-only content in a required field counts as empty.
- Cap free-text `maxLength` to the backend column length via `inputOptions={{ maxLength: N }}` on `InputField`. The native input silently truncates paste. See [Forms](#forms) in the top-level rules.
- If the max length is unusual (e.g. a 48-character password), show an inline error on the field instead of relying on silent truncation.
- Autofill counts as user interaction — treat autofilled fields as touched.

#### In-flight and submission lifecycle

- During submission, the submit button shows a spinner AND is disabled. Do not change the button's color/variant.
- Form fields are disabled while a submission is in flight — the user cannot edit during the request.
- The submit handler must guard against a second submission while one is in flight. Do not rely solely on the button being disabled.
- The Cancel button remains enabled during submission and closes the modal immediately. It does not abort the in-flight request; the request completes in the background.
- On success, fire the success toast BEFORE navigation (see [Notifications](../../.claude/rules/fleet-frontend.md#notifications) and #48088) and close the modal.
- On failure, fields become editable again, the submit button re-enables immediately, and server errors surface per [Server-side errors](#server-side-errors).
- Closing a modal with unsaved changes silently discards them. No confirmation dialog. (Exceptions like the SQL editor stay exceptions.)

#### Visual affordances

- There is no visual indicator for required fields. No asterisk, no `(required)` suffix. Users discover requirements through post-interaction errors.
- On error, `FormField` renders the error text in the label slot, replacing the label text and applying the `--error` modifier (red).
- Help text below the input is independent of error state. Do not duplicate error messages into help text.
- The input border is red while an error is showing and returns to the default (black) when the error clears. There is no green "valid" transition.
- No inline error icon. Text only.

<!-- Design may iterate on error placement (e.g. moving the error text out of the label slot). Document any change here first. -->


#### Error message copy register

Every validation error follows a single grammar pattern:

**Verb + object + constraint (if any).**

- **Verb**: the action that fixes the error. `Enter`, `Choose`, `Select`, `Upload`.
- **Object**: the thing being fixed. `your email`, `a valid URL`, `a password`.
- **Constraint**: only when the rule isn't obvious. `with at least 8 characters`, `between 1 and 100`.

Examples:
- `Enter your email` — empty field
- `Enter a valid email` — bad format
- `Enter a password with at least 8 characters` — rule violation
- `Choose an end date after the start date` — logical conflict
- `Upload a file smaller than 5 MB` — limit

Rules:
- Second-person imperative, implied subject. Never `You must...` or `The user should...`.
- Present tense, active voice. Not `must be completed`, not `was not provided`.
- Article discipline: `your` for the user's own data (`your email`, `your name`); `a` for a value the user is constructing (`a valid URL`, `a password`).
- One sentence per error. If it needs a second sentence, the constraint probably belongs in help text below the field, not in the error.
- **No terminal periods.** The error renders in the label slot, and labels don't end with periods.
- Use `fleet` not `team` in new copy. The codebase still uses `team_id` etc.; that stays. See [Terminology](../../.claude/rules/fleet-frontend.md#terminology).

For errors the user can't fix by editing the field — server failures, timeouts, network errors — use a different register: **what happened + what to do**. Example: `Couldn't save your changes. Try again in a few minutes.` This is the one place where periods appear (two sentences).

## Tier modes

The UI changes shape in two licensing-related modes. Both can be active at the same time on a Premium Primo tenant, so most gates need to consider them independently:

- **Fleet Free** — the free tier. Many features are hidden, paywalled with `<PremiumFeatureMessage />`, or restricted. Flag: `!isPremiumTier` (the context value is `true` for Premium / Enterprise, `false` for Free, so the Free check is the negation).
- **Primo mode** — a single-fleet Premium installation for partner deployments. The fleet switcher is hidden, fleet creation is disabled, and empty-state copy collapses. Flag: `isPrimoMode` (derived from `config.partnerships.enable_primo`).

Both flags can be true simultaneously (a Premium Primo tenant), and `isPrimoMode` is computed locally per-component, not in `AppContext` — see Gotchas. `frontend/components/CommandPalette/` is the canonical reference for handling both flags in tandem.

### Fleet Free

#### How to check

```tsx
const { isPremiumTier } = useContext(AppContext);
if (!isPremiumTier) {
  // Fleet Free behavior
}
```

`isPremiumTier` is derived from `config.license.tier === "premium"` in `utilities/permissions/permissions.ts` and set once during `AppContext` initialization — `true` for Premium / Enterprise, `false` for Free. **The Free check is the negation, `!isPremiumTier`.** Tests inject the value via the `app` context override.

#### Patterns

**Early-return paywall** — for full-page premium features. Use `<PremiumFeatureMessage />` (`components/PremiumFeatureMessage/`); don't roll your own:

```tsx
if (!isPremiumTier) {
  return <PremiumFeatureMessage />;
}
// render the feature
```

**Conditional list inclusion** via array spread — for items in palettes, menus, tables, etc.:

```tsx
...(isPremiumTier
  ? [{ id: "add-fleet-maintained-app", label: "Add Fleet-maintained app", ... }]
  : [])
```

For other patterns (conditional props on sub-components, React Query `enabled` to defer premium-only fetches, hidden table columns), see `frontend/components/CommandPalette/` and `pages/hosts/details/cards/Software/` as the canonical catalogs.

### Primo mode

Primo mode is a partnership mode (`partnerships.enable_primo`) that provides a single-fleet experience. It hides fleet creation, defaults to the "Unassigned" fleet context, and simplifies empty states. It can be active at the same time as Fleet Free *or* Premium, and at the same time as GitOps mode — these all affect different things.

#### How to check

```tsx
const { config } = useContext(AppContext);
const isPrimoMode = config?.partnerships?.enable_primo || false;
```

There's also a `PRIMO_TOOLTIP` constant in `utilities/constants.tsx` for disabled tooltips.

#### What it affects

- **"Create fleet" button**: disabled on ManageFleetsPage
- **Fleet switcher**: hidden (both the page `TeamsDropdown` header and the command palette fleet picker)
- **Selected fleet**: `useTeamIdParam` defaults to "Unassigned" instead of "All fleets"
- **Empty states**: skip the fleet-scoped copy premium normally shows, falling back to the generic header that free tier already uses (e.g., "No policies yet" instead of "No policies for this fleet" or "No policies apply to all fleets")
- **User form**: fleets dropdown disabled
- **Software automations**: accessible from "Unassigned" (normally only from "All fleets")

#### Pattern: disable with tooltip

The shared `Button` component doesn't accept a tooltip prop — use one of these patterns instead.

**`TableContainer` action buttons** accept `disabledTooltipContent` directly on the
`actionButton` config (see `ManageFleetsPage.tsx`):

```tsx
const disabledTooltip = isPrimoMode ? PRIMO_TOOLTIP : null;

<TableContainer
  // ...
  actionButton={{
    name: "create fleet",
    buttonText: "Create fleet",
    onClick: toggleCreateFleetModal,
    disabledTooltipContent: disabledTooltip,
  }}
/>;
```

**Standalone controls** (a `Button`, `Radio`, etc.) should be wrapped in
`TooltipWrapper` and disabled explicitly:

```tsx
<TooltipWrapper tipContent={PRIMO_TOOLTIP} disableTooltip={!isPrimoMode} showArrow>
  <Button disabled={isPrimoMode} onClick={onCreate}>
    Create fleet
  </Button>
</TooltipWrapper>
```

`RevealButton` is the exception — it accepts `disabledTooltipContent` directly.

### Testing both modes

Mock the flag in the test context (`isPremiumTier: false` for Free; `config: createMockConfig({ partnerships: { enable_primo: true } })` for Primo) and assert presence / absence. The established structure (from `CommandPalette/helpers.tests.ts`) is a top-level `describe` block per mode. Add at least one assertion per gate the feature touches.

### Gotchas

1. **`isPrimoMode` is not in AppContext.** Every component derives it from `config?.partnerships?.enable_primo`. If your refactor moves code out of a component that already had `config` in scope, you'll lose the Primo gate silently.
2. **Both flags can be true.** A Premium Primo tenant. Gates that test only one (e.g., `if (!isPremiumTier) hide` or `if (isPrimoMode) hide`) are usually incomplete.
3. **Dual-gated UI is common.** Table columns, picker columns, and entity selectors often gate on `isPremiumTier && !isPrimoMode` — premium-only AND multi-fleet-only. If your feature falls in either bucket, the dual gate is probably the right shape.
4. **`useTeamIdParam` defaults change in Primo.** "Unassigned" instead of "All fleets" for the global default. If your code reads `currentTeamId === -1` to mean "all fleets selected," that path won't fire in Primo.

### How tier modes differ from GitOps mode

| | Fleet Free | Primo mode | GitOps mode |
|---|---|---|---|
| Purpose | Free vs Premium feature gating | Single-fleet UI for partners | Repository-driven config management |
| Config | `license.tier !== "premium"` (i.e., `!isPremiumTier`) | `partnerships.enable_primo` | `gitops.gitops_mode_enabled` |
| Main effect | Paywall (`<PremiumFeatureMessage />`) | Disables fleet creation, defaults to "Unassigned" | Disables manual editing |
| Component | Conditional checks + paywall | Conditional checks on the flag | `GitOpsModeTooltipWrapper` |

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

### Custom hooks

Along with the hooks supplied by React such as `useEffect()` and `useState()`, you may create custom hooks as needed. A custom hook is a shared helper function that uses React state internally, for example to extract certain properties from a context or update a value when state changes.  A good example of a widely-used custom hook is `useTeamIdParam`, which returns information about the currently selected fleet and ensures (via its use of `useEffect()`) that the caller will get up-to-date values whenever a different fleet is selected.

Custom hook names should be camel-cased and use the `use` prefix, and should live in the `frontend/hooks` directory.

Current custom hooks include:

- [`useCheckTruncatedElement`](../hooks/useCheckTruncatedElement.ts) — Returns whether a referenced element's content is overflowing/truncated, updating on resize.
- [`useCheckboxListStateManagement`](../hooks/useCheckboxListStateManagement.tsx) — Manages checked/unchecked state for a list of policies with a toggle updater.
- [`useDeepEffect`](../hooks/useDeepEffect.ts) — `useEffect` variant that does a deep (lodash `isEqual`) comparison of dependencies.
- [`useGitOpsMode`](../hooks/useGitOpsMode.ts) — Reports whether GitOps mode is enabled (optionally per-entity, honoring exceptions) and returns the repo URL.
- [`useIsMobileWidth`](../hooks/useIsMobileWidth.tsx) — Tracks whether the viewport is below the 768px mobile breakpoint via `matchMedia`.
- [`usePlatformCompatibility`](../hooks/usePlatformCompatibility.tsx) — Debounced check of a SQL string for queryable-platform compatibility, with getter and renderer.
- [`usePlatformSelector`](../hooks/usePlatformSelector.tsx) — Manages platform-checkbox state (darwin/windows/linux/chrome) and renders the platform selector UI.
- [`useQueryTargets`](../hooks/useQueryTargets.ts) — `react-query` wrapper that fetches and groups target hosts/labels/teams for a query.
- [`useSoftwareInstallerMeta`](../hooks/useSoftwareInstallerMeta.ts) — Derives normalized metadata (installer type, FMA/VPP/Android flags, permissions, GitOps state) from a software title.
- [`useTeamIdParam`](../hooks/useTeamIdParam.ts) — Reads/writes the `fleet_id` URL param, resolving the current fleet and handling param strip/replace rules on fleet change.
- [`useToggleSidePanel`](../hooks/useToggleSidePanel.ts) — Simple open/closed state with toggle and explicit setter for a side panel.

## React context

[React context](https://reactjs.org/docs/context.html) is a way to share values across the
component tree. Use context for app-wide state or derived UI state that multiple components
need. For server state, use React Query for fetching, caching, and synchronization; some
server-derived values may still be exposed through context after they are fetched or
initialized. View currently working contexts in the [context directory](../context).

```typescript
// Consuming a context — destructure what you need from useContext
const { currentUser, isPremiumTier } = useContext(AppContext);
```

### Context catalog

| Context | Purpose | Use this when |
|---|---|---|
| `AppContext` | Global app state: current user, config, team selection, role flags, license info | You need user identity, permissions, feature flags, or the active fleet |
| `PolicyContext` | In-progress policy editing state: name, query, resolution, platform, labels | You're on the policy edit/create flow and need to persist form state across steps |
| `QueryContext` | In-progress report editing state: name, query body, frequency, targets, logging | You're on the report edit/create flow and need to persist form state across steps |
| `RoutingContext` | Stores a redirect location for post-auth navigation | You need to redirect the user after login (e.g., deep link they hit while logged out) |
| `TableContext` | Coordinates table row selection resets across components | You need to clear selected rows in a data table after a bulk action |

## Fleet API calls

### Making API calls

The [services](../services) directory stores all API calls and is to be used in two ways:

- A direct `async/await` assignment
- Using `react-query` if requirements call for loading data right away or based on dependencies.

Examples below:

#### Direct assignment

```tsx
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
      // maybe trigger notify.error
    }
  };

  return (
    // ...
  );
};
```

#### React Query

[react-query](https://react-query.tanstack.com/overview) is a data-fetching library that
gives us the ability to fetch, cache, sync and update data with a myriad of options and properties.

```tsx
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

##### Query keys

The `queryKey` must list every parameter that the `queryFn` passes to the API. The `QueryClient` is a singleton shared across the app, so any parameter missing from the key causes cross-entity cache bleed (for example, data fetched for team A being served to team B).

Rules:
- Always use an array, even when there are no parameters — `useQuery(["me"], ...)`, not `useQuery("me", ...)`.
- Every argument the `queryFn` forwards to the API must also appear in the key.

Example:

```ts
useQuery(
  ["aggregateProfileStatuses", teamId], // teamId is in the key...
  () => mdmAPI.getProfilesStatusSummary(teamId) // ...because the API call receives it
);
```

### Handling API errors

We pull the logic for handling error message into a `getErrorMessage` handler that lives in a sibling
`helpers.tsx` or `helpers.ts` file. This allow us to encapsulate the code for getting and formatting
the API error message away from the component. This will keep put components cleaner and easier
to read.

```tsx
/* In the component making a request */

try {
  await softwareAPI.install()
  // successful messgae
} catch (e) {
  notify.error(getErrorMessage(e))
}

/* in helpers.tsx */

// This function is used to abstract away the details of getting and formatting
// the error message we recieve from the API
export const getErrorMessage = (e: unknown) => {
  ...

  // return a string or a JSX.Element
  return "some error message"
}
```

## Page routing

We use React Router directly to navigate between pages. For page components,
React Router (v3) supplies a `router` prop that can be easily accessed.
When needed, the `router` object contains a `push` function that redirects
a user to whatever page desired. For example:

```tsx
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

## Command palette

The command palette is the keyboard surface for navigation and global actions. Power users discover features through it, so a missing entry is easy for them to overlook — but only the right kinds of features belong here.

Source: `frontend/components/CommandPalette/`. Items are defined per group under `groups/`.

### What belongs (and what doesn't)

**Belongs in the palette:**

- Navigation to any destination in the app — either its own top-level palette entry or nested under a parent entry via `subItems`
- Global create actions where no entity is pre-selected ("Add report" opens a blank form)
- Singleton config actions where the entity is implicit ("Edit Apple MDM" — there's only one Apple MDM config)
- Picker actions that open an in-palette search for the user to choose an entity (e.g., "View host")

**Doesn't belong:**

- Per-entity edit / delete operations (editing a specific label, deleting a specific host) — those live on the entity's row or detail page where the entity is already in scope
- Bulk-select operations that depend on an existing selection on a page
- One-off UI affordances (toggles, expanders) tied to a specific view

The dividing line: if the action requires the user to first pick a specific row, it stays on that row. If the action is global, a singleton, or starts a picker, it goes in the palette.

### When to add a palette entry

| Adding... | Goes in |
|---|---|
| A new top-level page (routed under a top nav item) | `groups/pages.ts` |
| A new global create action (modal / form / blank create page) | `groups/commands.ts` |
| A new picker action (like "View host") | `groups/commands.ts` with `opensPickerPage: true`, plus a picker in `frontend/components/CommandPalette/components/` |
| A new MDM platform or connector (turn-on / singleton-edit) | `groups/mdm.ts` |
| A new automation hook | `groups/automations.ts` |
| A new settings page or admin route | `groups/settings.ts` |
| A new control / policy / script feature | `groups/controls.ts` |
| A new software action or view | `groups/software.ts` |

Nested destinations under an existing palette entry live in that entry's `subItems` array, not as top-level entries. The user reaches the sub-item by expanding the parent (chevron) or when their search promotes the sub-item into Best match.

These three "sub-" terms each mean exactly one thing in this codebase — keep them distinct:

- **Sub-item** — an `ICommandSubItem` in a parent palette entry's `subItems` array
- **Picker page** — the secondary screen opened when an entry has `opensPickerPage: true` (View host, View report, Switch fleet)
- **Sub-route** — an app route nested under another (e.g., `/settings/integrations` under `/settings`)

### Required and optional fields

```ts
interface ICommandItem {
  id: string;                // unique kebab-case
  label: string;             // sentence case, verb first ("Add report")
  group: typeof GROUPS[number]; // one of `GROUPS` in helpers.ts
  path?: string;             // navigation target (use withTeamId() if team-scoped)
  onAction?: () => void;     // alternative to path for custom side effects
  keywords?: string[];       // synonyms + aliases — see below
  teamName?: string;         // chip shown when the action switches the user's fleet context
  subItems?: ICommandSubItem[];
  opensPickerPage?: boolean;    // shows the chevron-right; required for picker actions
}
```

### Label conventions

- Sentence case: "Add report", not "Add Report".
- Verb first for actions: "Add", "Edit", "Delete", "Run", "View", "Manage", "Turn on" / "Turn off".
- No trailing punctuation.
- Match the destination page's own primary-button text where possible.
- Use **fleet** / **report** (current product terminology), not **team** / **query**. Existing items haven't been mass-renamed; this applies to *new* items only.

### Keyword authoring

Best match scoring is **label-first by tier.** `scoreMatch()` in `helpers.ts` ranks a single text (label or keyword) against the query and returns one of these tier values:

| Tier | Label score | Keyword score |
|---|---|---|
| exact | 100 | 50 |
| prefix | 90 | 40 |
| word-prefix | 80 | 30 |
| substring | 70 | — (label-only) |

Any label hit outranks any keyword hit — even the weakest label tier (substring, 70) beats the strongest keyword tier (exact, 50). That's what shapes how keywords should be written: they only matter when the query doesn't hit the label at all. Duplicating label words in keywords just adds a redundant, lower-scoring path.

A few additional behaviors worth knowing — see `computeBestMatch()` and `scoreItemForBestMatch()` in `helpers.ts` for the full mechanics:

- **Noise floor.** 2-character queries only consider label-exact + label-prefix (no word-prefix, no substring, no keywords). 3+ characters unlocks the full ladder. See `BEST_MATCH_MIN_QUERY` / `BEST_MATCH_FULL_LADDER_MIN`.
- **Multi-token search.** A query like "settings org" is also scored as two tokens; each must find a positive match (against label or keywords), and the item takes the *minimum per-token score*. This lets order-independent searches like "settings org" → "Organization settings" promote without a phrase match.
- **Word splits.** Word boundaries split on whitespace AND hyphens, so "API-only user" yields `["api", "only", "user"]` — a query for `only` word-prefix-matches.
- **Substring is label-only.** Keyword substrings don't score (too noisy with short tokens); keywords cap at word-prefix.

**Do:**

- Add single distinct words a user might type that aren't already in the label
- Add the standard verb synonyms for every action label:
  - `add` → `create`, `new`
  - `edit` → `update`, `change`, `modify`
  - `delete` → `remove`
  - `view` → `open`, `show`
  - `run` → `execute`
  - `turn on` → `activate`, `set up`, `configure`
- Add acronyms and alternate names users actually type: `idp`, `ca`, `cve`, `fma`, `abm`, `vpp`, `mdm`, `dep`, `ade`
- Add platform aliases where relevant:
  - Apple → `iphone`, `ipad`, `macbook`
  - Windows → `pc`, `win10`, `win11`
  - Android → `phone`, `tablet`
- Include legacy product terms during rename windows (e.g., `queries`, `query` on Reports until the term fully drains)

**Don't:**

- Repeat words from the label. `Add user` already scores "add" or "user" via the label tiers (exact / prefix / word-prefix / substring, 70–100). Adding them as keywords would only score lower (30–50), never changing the ranking.
- Use multi-word keyword phrases when a single word works. A keyword like `create` matches as keyword-exact / -prefix / -word-prefix at the token level. A multi-word keyword like `create user` only matches when the whole phrase appears as one token in the query — multi-token splitting won't reach into it.
- Pile in low-signal substrings ("the", "some", generic verbs).

### Permission gating

Mirror the destination page's gate exactly. If the page rejects technicians, gate the palette item on `!isTechnician`. If the destination renders `<PremiumFeatureMessage />` on free tier, gate the palette item on `isPremiumTier`. The palette must not route users to a screen they can't use.

Reuse existing flags from `ICommandPaletteContext` (`frontend/components/CommandPalette/helpers.ts`) — that interface is the source of truth for the full list. The flags fall into a few buckets:

- **Role-based write gates:** `canWrite`, `canAccessSettings`, `canAccessControls`, `canRunLiveReport`, `canAddSoftware`, `canEditCustomVariable`, `canManagePolicyAutomations`, `canManageSoftwareAutomations`, `canManageReportAutomations`, `isTechnician`
- **Tier / mode:** `isPremiumTier`, `isPrimoMode`, `isDarkMode`
- **Feature configured:** `isMacMdmEnabledAndConfigured`, `isWindowsMdmEnabledAndConfigured`, `isAndroidMdmEnabledAndConfigured`, `isVppEnabled`
- **Context shape:** `hasTeamSelected`, `currentTeam`, `availableTeams`, `config`, `search`

Add a new flag to `ICommandPaletteContext` + `CommandPalette.tsx` only when no existing one models the destination's check. When you add one, mirror the destination page's predicate exactly — several existing flags (`canManageReportAutomations`, `canEditCustomVariable`, `canAddSoftware`) document the narrower role checks they encode; follow that pattern.

### Team context (`teamName`)

Set `teamName` when invoking the action will switch the user's current fleet context. The palette renders it as a chip on the right so the user sees the upcoming switch before they click.

Each group builder receives an `IDerivedContext` (computed once by `deriveContext()` in `groups/derivations.ts`) as its second argument. Destructure the chip helper you need from it rather than hardcoding fleet names:

```ts
const buildExampleItems = (ctx, derived) => {
  const { switchesFromUnassigned, switchesFromAllFleets } = derived;
  // ...
};
```

The three chip helpers:

- `switchesFromUnassigned` — destination requires a specific fleet, action invokable from Unassigned
- `switchesFromAllFleets` — destination requires a specific fleet, action invokable from All fleets
- `defaultDestination` — destination always lands on the default (e.g., "All fleets")

Each returns `undefined` when no switch will actually happen, so you can pass it straight to `teamName` without guarding.

### Search-only items

Some entries are gated on the search string itself (e.g., the "Packs" page only appears when searching for `packs`). Use the `search` field from `ICommandPaletteContext` and a regex test:

```ts
.../packs|create new pack/.test(search.toLowerCase())
  ? [/* the item */]
  : []
```

Use this pattern sparingly — it bypasses the normal Best match ranking and should be reserved for legacy / deprecated features users only reach by name.

### Tests

Extend `frontend/components/CommandPalette/helpers.tests.ts` when adding a meaningful item:

- New page / command: assert it appears for the right roles, hides for the wrong ones
- Premium-only: assert it's absent in the `Fleet Free (isPremiumTier: false)` describe block
- Primo mode hidden: add to the `Primo Mode (isPrimoMode: true)` block
- New `teamName` chip: assert it renders / doesn't render against the relevant fleet contexts

The scoring helpers (`scoreMatch`, `scoreItemForBestMatch`, `computeBestMatch`, `highlightMatches`) and tier constants (`SCORE_LABEL_*`, `SCORE_KEYWORD_*`) have their own describe blocks in `helpers.tests.ts` — you don't need to re-test the framework when adding an item. If your new item exposes a specific ranking case worth pinning (e.g., a multi-token query that should promote it over a similarly-named item), add a small `computeBestMatch` test alongside.

## Styles

Below are a few need-to-knows about what's available in Fleet's CSS:

### Spacing

Prefer `gap` over `margin` for spacing between sibling elements when they share a flex or grid parent (otherwise `gap` has no effect). We have layout mixins in
`frontend/styles/var/mixins.scss` for common flex column patterns:

| Mixin | Gap | Use for |
|---|---|---|
| `vertical-page-layout` | 24px | Top-level page content |
| `vertical-card-layout` | 24px | Settings cards, OS settings panels |
| `vertical-form-layout` | 24px | Form field groups |
| `vertical-modal-layout` | 24px | Modal body content |
| `vertical-page-tab-panel-layout` | 24px | Tab panel content |
| `vertical-data-set-layout` | 16px | Definition lists, key-value field sets |

All use `flex-direction: column`; the 24px value is `$gap-page-component`. For arbitrary
spacing without a semantic name, use `flex-column-16px-gap` or `flex-column-32px-gap`.

### Modals

1) When creating a modal with a form inside, the action buttons (cancel, save, delete, etc.) should
   be wrapped in the `modal-cta-wrap` class to keep unified styles.

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

The recommended line limit per page/component is 500 lines. This is only a recommendation.
Larger files are to be split into multiple files if possible, especially components which could be split into subcomponents.

## Testing

At a bare minimum, we make every effort to test that components that should render data are doing so
as expected. For example: `HQRTable.tests.tsx` tests that the `HQRTable` component correctly renders
data being passed to it.

At a bare minimum, critical bugs released involving the UI will have automated testing discussed at the critical bug post-mortem with a frontend engineer and an engineering manager. We make every effort to add an automated test to either the unit, integration, or E2E layer to prevent the critical bug from resurfacing.

## Security considerations

### Sanitization for `dangerouslySetInnerHTML`

We make every effort to avoid using the `dangerouslySetInnerHTML` prop. When the
prop is necessary, sanitize any user-defined input with one of the approved
helpers below.

- `DOMPurify.sanitize(html, options)` — for rendering arbitrary HTML. Configure
  allowed tags and attributes explicitly at the call site, for example
  `{ ADD_ATTR: ["target"] }`.
- [`syntaxHighlight(value)`](../utilities/helpers.tsx) — for rendering JSON or
  code previews. The function HTML-escapes `&`, `<`, and `>` in the
  JSON-stringified output before wrapping tokens in `<span class="…">`, so the
  only markup it emits is a span with a fixed set of class names: `key`,
  `string`, `number`, `boolean`, `null`. Pass a value to JSON-serialize (object,
  array, or primitive), not a pre-built string of user content. The function
  calls `JSON.stringify` on its input before highlighting, so a pre-built
  string defeats the purpose of using it. See `ExampleWebhookUrlPayloadModal`
  or `HostStatusWebhookPreviewModal` for representative call sites.
- [`ClickableUrls`](../components/ClickableUrls/ClickableUrls.tsx) — for
  rendering user-provided text that may contain URLs as clickable links. It
  replaces URL-looking substrings with anchors (`target="_blank"` and
  `rel="noreferrer"`), then sanitizes the resulting HTML with
  `DOMPurify.sanitize` (assumes the input is text, not pre-built HTML) before
  rendering. Use it instead of constructing anchor HTML by hand when the source
  is user-provided text.

### Other frontend security considerations

Beyond the sanitization rules above, the following considerations apply across
the frontend.

Today:

- URL handling: treat any user-controlled value used in `href`, `window.open`,
  or `router.push`/`router.replace` as untrusted. Route to a known internal path
  and pass user data as URL params rather than building a full URL from a user
  value.
- Logging: `no-console` is disabled to support local debugging, but clean up
  `console.log` statements before merging. Do not log access tokens, session
  identifiers, or full response bodies that may contain user PII.
- Dependency review: when a pull request adds an entry to `package.json`,
  confirm the package is well-maintained, that the name is not a typosquat of a
  more popular package, and that the version pinned is consistent with the
  surrounding ecosystem.

Candidates for future work:

- Add an ESLint rule or `CODEOWNERS` gate on new uses of
  `dangerouslySetInnerHTML` so a reviewer must consciously opt into each new
  occurrence.
- Inventory existing uses of `dangerouslySetInnerHTML` and migrate any that can
  render through React directly.
- Reduce direct `localStorage` writes for values with a security implication.
  Prefer in-memory state for short-lived sensitive values.

### Frontend pitfalls in AI-assisted code

Fleet engineers are expected to use AI coding tools as part of their daily
workflow (see [AI tooling](https://fleetdm.com/handbook/engineering#ai-tooling)
and [AI usage guidelines](https://fleetdm.com/handbook/company/communications#ai-usage-guidelines)).
You own the code you submit, AI-assisted or otherwise. The handbook covers the
general principle; this section lists frontend pitfalls that AI tools produce
often enough to warrant a deliberate check on every diff:

- `dangerouslySetInnerHTML` introduced for user-controlled content without an in-diff
  sanitizer. If the prop is added and the same change does not also pass the
  input through `DOMPurify.sanitize`, `syntaxHighlight`, or `ClickableUrls`,
  request a sanitizer before approving.
- Dynamic code execution: `eval`, `new Function`, and `setTimeout` or
  `setInterval` invoked with string arguments have no legitimate place in
  product code.
- New runtime dependencies. Flag any added entry in `package.json` for the
  dependency-review pass above, even when the surrounding change appears
  unrelated.
- Permissive URL construction in anchors, `window.open` calls, or router pushes
  that interpolate user-controlled values without going through the approved
  helpers above.

See [`.claude/rules/fleet-frontend.md`](../../.claude/rules/fleet-frontend.md)
for the LLM-targeted summary of these rules.

## Other

### Local states

Our first line of defense for state management is local states (i.e. `useState`). We
use local states to keep pages/components separate from one another and easy to
maintain. If states need to be passed to direct children, then prop-drilling should
suffice as long as we do not go more than two levels deep. Otherwise, if states need
to be used across multiple unrelated components or 3+ levels from a parent,
then the [app's context](#react-context) should be used.

### Reading and updating configs

If you are dealing with a page that *updates* any kind of config, set the local
config with the response of your update call to make sure it has the latest.

### Toast notifications

Use `notify.success(msg)` / `notify.error(msg, { response })` / `notify.batch([...])` from
`components/ToastNotification`. Success toasts auto-dismiss after 5s by default; error toasts are sticky by default.
Visible toasts are dismissed automatically on URL change.

**When showing a success toast and navigating, call `notify.success` before `router.push` / `router.replace`** — the reverse order can break auto-dismiss on the destination page (#48088).

```tsx
// first notify
notify.error("Something went wrong");
// then push
router.push(newPath);
```

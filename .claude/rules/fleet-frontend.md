---
paths:
  - "frontend/**/*.ts"
  - "frontend/**/*.tsx"
---

# Fleet Frontend Conventions

## Component Structure
Every component should have this 4-file structure:
- `ComponentName.tsx` — Main component
- `_styles.scss` — Component-specific SCSS styles
- `ComponentName.tests.tsx` — Tests
- `index.ts` — Named export

Use the component generator for new components:
```
./frontend/components/generate -n PascalCaseName -p optional/path/to/parent
```

## React Query
- Prefer `useQuery` over manual `useState`/`useEffect` for API data
- Use `useMutation` for write operations — invalidate related queries on success
- Use the `enabled` option to defer a query until its dependencies are ready

### Query keys

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

## API Services
- API clients live in `frontend/services/entities/`
- Use `sendRequest(method, path, body?, queryParams?)` from `frontend/services/`
- Endpoint constants in `frontend/utilities/endpoints.ts`
- Build query strings with `buildQueryStringFromParams()` from `frontend/utilities/url/`
- Build full paths with `getPathWithQueryParams(path, params)` — auto-filters undefined/null values

## Permission Checking
Use helpers from `frontend/utilities/permissions/permissions.ts`:
- Global roles: `permissions.isGlobalAdmin(user)`, `isGlobalMaintainer(user)`, `isOnGlobalTeam(user)`
- Team roles: `permissions.isTeamAdmin(user, teamId)`, `isTeamMaintainer(user, teamId)`, `isTeamObserver(user, teamId)`
- Multi-team: `permissions.isAnyTeamAdmin(user)`, `isOnlyObserver(user)`
- License: `permissions.isPremiumTier(config)`, `isFreeTier(config)`
- MDM: `permissions.isMacMdmEnabledAndConfigured(config)`, `isWindowsMdmEnabledAndConfigured(config)`

## Team Context
Use the `useTeamIdParam` hook for team-scoped pages:
- `currentTeamId`: -1 (All teams), 0 (No team), or positive team ID
- `teamIdForApi`: undefined (All teams), 0 (No team), or positive ID — **always use this for API calls**
- `handleTeamChange(newTeamId)` to switch teams
- `isTeamAdmin`, `isTeamMaintainer`, `isObserverPlus` for role checks

## Routing & URL state
Use react-router, not `window.location` / `window.history`. Direct window mutation desyncs react-router's location state.
- Read query params from `location.query`, not `URLSearchParams(window.location.search)`.
- Mutate the URL with `router.replace`/`router.push` or `browserHistory.replace`/`.push`, not `window.history.replaceState`.
- Auto-correcting a missing/invalid query param inside a `useEffect` MUST use `router.replace`, not `router.push`, so browser Back isn't trapped.
- Internal `<CustomLink>` (and any `router.push` / `<Link>`) to a fleet-scoped route MUST preserve the current fleet via `getPathWithQueryParams(PATHS.X, { fleet_id: teamId })`. Linking to the bare path drops fleet context and lands the user on the wrong fleet. Applies to any path that reads `fleet_id` from the query string (most `/software`, `/hosts`, `/policies`, `/queries`, `/controls` routes). `getPathWithQueryParams` filters undefined/null, so pass `teamId` directly — `fleet_id=0` (No team) is a valid, intentional value and must be preserved.

## Notifications
- Use `notify.success(msg)` / `notify.error(msg, { response })` / `notify.batch([...])` from `components/ToastNotification`.
- **When showing a success toast and navigating, call `notify.success` before `router.push` / `router.replace`** — the reverse order can break auto-dismiss on the destination page (#48088).
- Success toasts auto-dismiss after 5s by default; error toasts are sticky by default.

## XSS Prevention
- ALWAYS sanitize user-generated HTML before `dangerouslySetInnerHTML`. Approved helpers:
  - `DOMPurify.sanitize(html, options)` — arbitrary HTML. Configure allowed tags/attributes explicitly: `{ ADD_ATTR: ["target"] }`
  - `syntaxHighlight(value)` from `frontend/utilities/helpers.tsx` — JSON/code previews. Input must be a value to JSON-serialize (object/array/primitive), not a pre-built string of user content
  - `ClickableUrls` from `frontend/components/ClickableUrls/` — plain text that may contain URLs, rendered as clickable links
- See `frontend/docs/patterns.md#security-considerations` for full guidance, including frontend pitfalls common in AI-assisted code

## String Utilities
Use helpers from `frontend/utilities/strings/stringUtils.ts`:
- `capitalize(str)`, `capitalizeRole(role)` — handle special casing (Observer+)
- `pluralize(count, singular, pluralSuffix, singularSuffix)` — "1 host" vs "2 hosts"
- `stripQuotes(str)`, `strToBool(str)` — input parsing
- `enforceFleetSentenceCasing(str)` — respects Fleet stylization rules

## Software titles

### Display name
Render software title names via `getDisplayedSoftwareName(name, display_name)` from `pages/SoftwarePage/helpers.tsx` — never raw `t.name` or open-coded `display_name || name`. See `frontend/docs/patterns.md`.

### Icons
`<SoftwareIcon name={...}>` uses `name` for fallback icon matching when `icon_url` is null (Fleet-maintained apps depend entirely on this). Pass the **raw** `name`, never `getDisplayedSoftwareName(...)` or `display_name`, or admin renames will break the icon match. When a flattened row carries only one name field, add a sibling `iconName` (raw) field and feed THAT to `<SoftwareIcon>`. See `frontend/docs/patterns.md` and #47123.

## Styling (SCSS + BEM)
- Define `const baseClass = "component-name"` at the top of the component
- Elements: `` className={`${baseClass}__element-name`} ``
- Modifiers: `` className={`${baseClass}--modifier`} ``
- Use `classnames()` for conditional classes
- Style files use underscore prefix: `_styles.scss`
- Prefer `gap` over `margin` for spacing between sibling elements when the parent is `display: flex`/`grid`. Use the layout mixins from `frontend/styles/var/mixins.scss`: `vertical-card-layout`, `vertical-form-layout`, `vertical-modal-layout`, `vertical-page-layout`, `vertical-page-tab-panel-layout`, `vertical-data-set-layout`

## Forms
Cap free-text inputs' `maxLength` to the backend column length (check `server/datastore/mysql/schema.sql`, don't guess) via `inputOptions={{ maxLength: NAME_MAX_LENGTH }}` on `InputField`, using a local constant.

## Validation

See `frontend/docs/patterns.md#data-validation` for full rules and rationale. These describe target behavior; existing forms migrate one at a time as they're touched. A shared validation hook is planned to encode them.

### Error timing
- Never show a field's error before the user has interacted with it (typed into it, or blurred it while dirty).
- On blur of an interacted-with field, validate that field and show its error inline. Don't modify other fields' errors.
- On submit, show inline errors on every invalid field simultaneously, then return without submitting.
- Autofill counts as user interaction — treat autofilled fields as touched.
- On Edit forms, pre-filled invalid values don't show errors until interaction.

### Error clearing
- On focus (click-in) of an errored field, clear that field's error immediately — don't wait for a valid value.
- Re-validate on blur. Never validate on keystroke.
- Typing in one field never clears errors on other fields.
- When a validation becomes irrelevant (e.g. conditional requirement removed by toggling a checkbox), clear the now-irrelevant error immediately.

### Error priority
Presence errors take priority over format errors. Show one error per field at a time; never stack.

### Submit button state
- Enabled by default — do not disable for empty required fields, currently-invalid values, unchanged Edit forms, or prior server errors.
- Only disable while a submission is in flight.
- If the user clicks submit with invalid data, the submit handler shows all inline errors and returns without submitting.

### Server-side errors
- Field-specific server errors: render inline on the field AND fire a toast. Submit stays enabled.
- Global / cross-field server errors (e.g. `formatErrorResponse` `.base`): toast only.
- Clear a server-set error on next focus of that field — same rule as client-side.
- Backend field key → UI field key mapping stays local to the form until the API is normalized.

### Optional and disabled fields
- Empty optional fields: no error, submit enabled.
- Optional fields with a format constraint (e.g. optional email): show inline error on invalid, but never disable submit.
- Disabled fields skip validation entirely — never in an error state regardless of value.

### In-flight submission
- Submit button shows a spinner AND is disabled. Do not change the button's color/variant.
- Disable form fields too, not just the button.
- Guard the submit handler against double-submit; don't rely on the disabled button alone.
- Cancel button stays enabled during submission and closes the modal. It does not abort the in-flight request.
- Fire success toasts BEFORE navigation (see [Notifications](#notifications), #48088).
- Closing a modal with unsaved changes silently discards them.

### Input hygiene
- Trim leading/trailing whitespace client-side before submitting; send the trimmed value.
- Whitespace-only content in a required field counts as empty.
- Match DB column max via `inputOptions.maxLength` on `InputField` (see [Forms](#forms) above). Native silent truncation is the default. Show an inline error only when the limit is awkward (e.g. a 48-char password).

### Visual affordances
- No visual indicator for required fields — no asterisk, no `(required)` suffix.
- On error, `FormField` renders the error text in the field's label slot (replacing the label) with the `--error` modifier — red.
- Input border is red while an error is showing, black (default) when clear. No green "valid" transition.
- No inline error icon; text only.

### Copy register
- Grammar: `Verb + object + constraint`. Second-person imperative. Active voice. Present tense. One sentence per error.
- Verb picks: `Enter`, `Choose`, `Select`, `Upload`.
- Article discipline: `your` for the user's own data (`your email`); `a` for a value they're constructing (`a valid URL`, `a password`).
- **No terminal periods** — the error renders where the label would be, and labels don't end in periods.
- `Enter your email`, not `Email is required` / `Email field must be completed`.
- Use "fleet" not "team" in error strings.
- For non-fixable errors (server failure, timeout, network): use `<what happened>. <what to do>.` — two sentences, periods allowed. Example: `Couldn't save your changes. Try again in a few minutes.`

## Lists & rows
User-typed free-text fields (`name`, `title`, `label`, `description`) inside an `UploadList` `ListItemComponent`, a `__row` flex container with sibling actions/badges, or a `TableContainer` open-text cell — wrap the value in `<TooltipTruncatedText value={...} />` and give the immediate parent `flex: 1; min-width: 0`.

Anti-pattern:
```tsx
<span className={`${baseClass}__row-name`}>{item.name}</span>
```

## Interfaces & Types
- Interface files live in `frontend/interfaces/` with `I` prefix: `IHost`, `IUser`, `IPack`
- Legacy pattern: some files export both PropTypes (default export) and TypeScript interfaces (named export)
- New code should use TypeScript interfaces only
- API interface naming: use `*FormData` for form-driven request bodies, `*ApiParams`/`*QueryParams` for request params, `*Response` for API responses, `*QueryKey` when typing a React Query key. Avoid `*Body`, `*PostBody`, `*Payload`, `*Request` for API request bodies. `*PreviewPayload` is fine for outgoing webhook shapes (matches the "Preview payload" UI terminology).

## Hooks & Context
- Custom hooks in `frontend/hooks/` — e.g., `useTeamIdParam`, `useCheckboxListStateManagement`
- Context providers in `frontend/context/` — `AppContext` for global state, `NotificationContext` for flash messages

## Tier modes (Fleet Free + Primo mode)
Load the `tier-modes` skill when:
- **Adding a new top-level page, feature page, or significant UI surface** (modal, side panel, dashboard, settings section, new tab) — for the end-of-task gap check on whether Free / Primo behavior was decided.
- **Introducing NEW tier gating to code that doesn't have it yet** — to follow the established gating patterns.

Editing inside already-gated code (adding a field to a premium-only form, fixing a bug in a paywalled flow) doesn't need this — the tier decision is already made there.

## Terminology
- "Teams" are now called "fleets" in the product. Code still uses `team_id`, `useTeamIdParam`, `permissions.isTeamAdmin`, etc. — don't rename existing APIs, but use "fleet" in new user-facing strings and comments.
- "Queries" are now called "reports." The word "query" now refers solely to a SQL query. Code still uses `useQuery`, `queryKey`, etc. for React Query — that's unrelated to the product terminology change.

## Command palette
If you edit `frontend/router/paths.ts` or `frontend/router/index.tsx`, add a new MDM connector / singleton config, add a new global create / automation / settings action, or add a new picker action, load the `command-palette` skill before finishing — these changes almost always need a matching entry under `frontend/components/CommandPalette/groups/`. The palette is for navigation and global actions — not per-entity (row-level) operations, bulk-select actions, or per-view UI toggles.

## Linting & Formatting
- ESLint: extends airbnb + typescript-eslint + prettier
- Prettier: default config (`.prettierrc.json`)
- `console.log` is allowed (`no-console` is off) — useful for debugging, but clean up before merging
- `react-hooks/exhaustive-deps` is enforced as a warning — include all dependencies in hook dependency arrays
- Run `make lint-js` or `yarn lint` and `npx prettier --check frontend/` before submitting

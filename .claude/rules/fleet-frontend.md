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

## Notifications
- Use `renderFlash(alertType, message)` from `NotificationContext`
- Types: `"success"`, `"error"`, `"warning-filled"`
- Use `renderMultiFlash()` for batch operations

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

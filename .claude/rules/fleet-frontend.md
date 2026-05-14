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

## Notifications
- Use `renderFlash(alertType, message)` from `NotificationContext`
- Types: `"success"`, `"error"`, `"warning-filled"`
- Use `renderMultiFlash()` for batch operations

## XSS Prevention
- ALWAYS sanitize user-generated HTML with `DOMPurify.sanitize(html, options)` before `dangerouslySetInnerHTML`
- Configure allowed tags/attributes explicitly: `{ ADD_ATTR: ["target"] }`

## String Utilities
Use helpers from `frontend/utilities/strings/stringUtils.ts`:
- `capitalize(str)`, `capitalizeRole(role)` — handle special casing (Observer+)
- `pluralize(count, singular, pluralSuffix, singularSuffix)` — "1 host" vs "2 hosts"
- `stripQuotes(str)`, `strToBool(str)` — input parsing
- `enforceFleetSentenceCasing(str)` — respects Fleet stylization rules

## Styling (SCSS + BEM)
- Define `const baseClass = "component-name"` at the top of the component
- Elements: `` className={`${baseClass}__element-name`} ``
- Modifiers: `` className={`${baseClass}--modifier`} ``
- Use `classnames()` for conditional classes
- Style files use underscore prefix: `_styles.scss`

## Interfaces & Types
- Interface files live in `frontend/interfaces/` with `I` prefix: `IHost`, `IUser`, `IPack`
- Legacy pattern: some files export both PropTypes (default export) and TypeScript interfaces (named export)
- New code should use TypeScript interfaces only

## Hooks & Context
- Custom hooks in `frontend/hooks/` — e.g., `useTeamIdParam`, `useCheckboxListStateManagement`
- Context providers in `frontend/context/` — `AppContext` for global state, `NotificationContext` for flash messages

## Terminology
- "Teams" are now called "fleets" in the product. Code still uses `team_id`, `useTeamIdParam`, `permissions.isTeamAdmin`, etc. — don't rename existing APIs, but use "fleet" in new user-facing strings and comments.
- "Queries" are now called "reports." The word "query" now refers solely to a SQL query. Code still uses `useQuery`, `queryKey`, etc. for React Query — that's unrelated to the product terminology change.

## Linting & Formatting
- ESLint: extends airbnb + typescript-eslint + prettier
- Prettier: default config (`.prettierrc.json`)
- `console.log` is allowed (`no-console` is off) — useful for debugging, but clean up before merging
- `react-hooks/exhaustive-deps` is enforced as a warning — include all dependencies in hook dependency arrays
- Run `make lint-js` or `yarn lint` and `npx prettier --check frontend/` before submitting

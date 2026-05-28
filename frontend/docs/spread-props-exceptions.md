# Spread-prop exceptions

`patterns.md` directs us to use explicit prop assignment (`<Foo a={a} b={b} />`) instead of object spread (`<Foo {...props} />`). The reasons are well known:

- It hides what is actually being passed to a component from the reader.
- It makes refactors brittle: adding a prop to the source bag silently changes the target's behavior.
- On native DOM elements it is a real security/correctness footgun. Anything the spread bag contains will be applied — including `dangerouslySetInnerHTML`, `href`, `src`, `srcDoc`, `formAction`, event handlers, ARIA attributes, etc. If any of those values can come from user-controlled input, you have an XSS / open-redirect / DOM-clobbering vector.

That said, the codebase has ~270 files that legitimately use `{...props}` / `{...rest}`. This document catalogs the categories that are accepted, why they are safe, and the rules that keep them safe. New code should fit into one of these categories or use explicit assignment.

## Quick decision tree

```
Is the value coming from user input, a URL, an API response, or markdown source?
├── Yes → DO NOT spread. Pick out the specific fields you need.
└── No  → Is the target a native DOM element (<a>, <img>, <iframe>, <object>,
         <form>, <script>, <link>, <source>, <embed>)?
         ├── Yes → DO NOT spread. Assign the specific attributes explicitly.
         └── No  → Spread is acceptable if it fits one of the categories below.
```

## Accepted exception categories

### 1. `react-select` custom subcomponents

**Where**: `components/ActionsDropdown`, `components/TeamsDropdown`, `components/top_nav/UserMenu`, `components/forms/fields/DropdownWrapper`, `pages/DashboardPage/cards/ActivityFeed/components/ActivityTypeDropdown`, `pages/hosts/ManageHostsPage/components/{CustomDropdownIndicator,CustomLabelGroupHeading,CustomValueContainer,LabelFilterSelect}`.

**Shape**:

```tsx
const CustomDropdownIndicator = (props: DropdownIndicatorProps<...>) => (
  <components.DropdownIndicator {...props} className={baseClass}>
    {/* ... */}
  </components.DropdownIndicator>
);
```

**Why accepted**: react-select 5's [custom components contract](https://react-select.com/components) requires the wrapper to forward the *full* internal props bag (`innerRef`, `innerProps`, `selectProps`, `cx`, `getStyles`, …) to the base `components.X`. Picking individual props is impractical (the surface is large and library-versioned) and any omission silently breaks keyboard handling, styling, or accessibility.

**Why it's safe**: `props` is constructed by react-select itself, not by user input. No DOM attribute in the bag is derived from anything an attacker can influence.

**Rules**:
- Spread only into `components.X` from `react-select`, not into a raw DOM element you render yourself inside the wrapper.
- If you need to override one prop (e.g. `className`), put it *after* the spread so it wins.

### 2. `react-table` v7 prop getters

**Where**: `components/TableContainer/DataTable/DataTable.tsx` (`getRowProps`, `getCellProps`, `getHeaderProps`), plus `*TableConfig.tsx` files that call `getToggleAllRowsSelectedProps()` to render the header/cell checkbox.

**Shape**:

```tsx
<td {...cell.getCellProps()}>{cell.render("Cell")}</td>
<Checkbox {...checkboxProps} enableEnterToCheck />
```

**Why accepted**: react-table's API *is* the prop-getter pattern — each getter returns the synthesized props (key, role, style, aria-*, event handlers) needed to wire that node into the table state machine. Re-implementing those by hand defeats the library.

**Why it's safe**: The bag is produced by react-table from the column/row config defined in our own `*TableConfig.tsx`. Data values from the API are rendered through `cell.render("Cell")` — they never become attributes on the `<td>`.

**Rules**:
- Spread the getter's return value onto the element the library expects (`<table>`, `<thead>`, `<tr>`, `<th>`, `<td>`) — not onto unrelated children.
- Never merge user-controlled data into the object before spreading.

### 3. `react-markdown` renderer overrides

**Where**: `components/FleetMarkdown/FleetMarkdown.tsx`.

**Shape**:

```tsx
code: ({ children, ...props }) => <code {...props}>{children}</code>;
```

**Why accepted**: react-markdown passes a known set of HAST-derived props (`className` for the language, `node`, etc.) to override renderers. Re-listing them by hand would couple us to react-markdown's internal prop list.

**Why it's safe**: react-markdown sanitizes by default and does not pass through arbitrary HTML attributes from the source markdown — the props bag here is library-controlled metadata, not raw user HTML. The actual user content arrives through `children`, where React escapes it.

**Rules**:
- Keep `dangerouslySetInnerHTML` and `rehype-raw` *off*. If either is ever enabled, the spread above becomes an XSS sink and must be replaced with explicit attribute passthrough plus DOMPurify.
- Do not extend this pattern to `a`, `img`, `iframe`, `video`, `source`, `link`, `script` — for those, pull out `href`/`src` and validate the scheme before rendering.

### 4. SVG icon components (software/brand icons)

**Where**: `pages/SoftwarePage/components/icons/*.tsx` (~265 files), e.g. `AcrobatReader.tsx`, `Miro.tsx`, `WindowsDefender.tsx`.

**Shape**:

```tsx
const Miro = (props: SVGProps<SVGSVGElement>) => (
  <svg xmlns="http://www.w3.org/2000/svg" width={32} height={32} {...props}>
    <image href="data:image/png;base64,..." />
  </svg>
);
```

**Why accepted**: These are leaf presentational components. The whole point is to let callers override `width`, `height`, `className`, `aria-label`, etc., without us re-declaring the full `SVGAttributes` surface (dozens of props) on every icon.

**Why it's safe**:
- The TypeScript signature `SVGProps<SVGSVGElement>` constrains callers to valid SVG attributes — there is no `href`, `src`, or `dangerouslySetInnerHTML` on `<svg>`.
- The embedded `href` data-URIs are *hardcoded* inside each file (checked into the repo), not derived from props. A caller cannot swap in a malicious payload via `{...props}`.
- Callers are internal (`IconMap`, dashboards) and always pass static or strongly-typed values — never raw API responses.

**Residual risk**: `dangerouslySetInnerHTML` is technically a valid prop on any HTML/SVG element via `HTMLAttributes`/`SVGAttributes`. A misuse like `<Miro dangerouslySetInnerHTML={{ __html: userInput }} />` would let the spread propagate it onto the `<svg>`. This is theoretical — TypeScript would still allow it but no caller in the codebase does this. If we ever lint for it, add `dangerouslySetInnerHTML` to a banned-prop list.

**Rules**:
- Keep the typed signature (`SVGProps<SVGSVGElement>`). Don't widen to `any` or `Record<string, unknown>`.
- The spread target must be the `<svg>` root, not an inner `<image>` or `<a>` element.
- Never accept the inner `href` / `xlink:href` from props.

### 5. Factory / builder helpers

**Where**: `test/test-utils.tsx` (`createWrapperComponent`), `components/Icon/Icon.tsx` (`<IconComponent {...props} />` after `useMemo` builds the bag), test render helpers in `*.tests.tsx` and `*.stories.tsx`.

**Shape**:

```tsx
// Icon.tsx
const props = useMemo(() => Object.assign({},
  color === undefined ? undefined : { color },
  size === undefined ? undefined : { size },
), [color, size]);
return <IconComponent {...props} />;

// test-utils.tsx
return ({ children }) => (
  <WrapperComponent {...props}>{...}</WrapperComponent>
);
```

**Why accepted**: The bag is *constructed in the same file* by code that owns the shape. The author can audit exactly what keys exist; the spread is just a terser equivalent of explicit assignment.

**Why it's safe**: The values are local variables or hook-derived state — never untrusted input. Tests and stories run in dev contexts and don't reach end users.

**Rules**:
- The object being spread must be built locally in the same component/helper. Spreading an object that came in as a prop or from a network call falls back to the strict rule.
- Don't spread onto a native element from a factory unless you can list every possible key (otherwise prefer explicit assignment).

### 6. Test/storybook props forwarding

**Where**: `**/*.tests.tsx`, `**/*.stories.tsx` — e.g. `render(<MyComponent {...props} />)`.

**Why accepted**: Tests and stories are not shipped to users. The "props" object is a fixture built in the test file. Forwarding it whole keeps test setup readable.

**Why it's safe**: Not production code.

**Rules**:
- Keep these spreads inside test/story files only. Do not spread test fixtures into production helpers.

## What is NOT accepted

- **Spreading API responses onto JSX.** A response shape can change server-side and silently start writing new DOM attributes. Destructure the fields you need.
- **Spreading onto navigation/embed elements** (`<a>`, `<img>`, `<iframe>`, `<object>`, `<embed>`, `<source>`, `<link>`, `<script>`, `<form>`, `<video>`, `<audio>`). Always pick out `href`/`src`/etc. and validate the scheme (`http(s):`, no `javascript:` or `data:` unless explicitly required).
- **Re-spreading anything that originated outside this repo's code** — markdown source, query strings, host facts, custom variables, software metadata, MDM payloads. Treat all of those as tainted.
- **Mixing trusted and untrusted data in the same bag and then spreading.** Once an untrusted field lands in the bag, the whole bag is untrusted.
- **`{...rest}` after destructuring user-controlled fields out.** The remaining keys may still include attributes you forgot about (e.g. `style`, `dangerouslySetInnerHTML`). Build a fresh, explicit object instead.

## Lint guidance (future work)

A targeted ESLint rule could enforce most of this:
- `react/jsx-props-no-spreading` allowlist for `components.*` (react-select), prop-getter call expressions (`get*Props()`), and locally-built objects.
- Ban `{...props}` on a fixed list of intrinsic elements: `a`, `img`, `iframe`, `object`, `embed`, `source`, `link`, `script`, `form`, `video`, `audio`.

Until then, reviewers should apply the decision tree at the top of this file.

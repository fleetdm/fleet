# CLAUDE.md — Fleet Website

This file provides guidance to Claude Code (claude.ai/code) when working with code in the `website/` folder.

## About

Sails.js 1.5.17 web application. Node.js 20+, PostgreSQL, EJS templates, Vue.js/Parasails frontend, LESS styles, Grunt build system.

## Architecture

```
api/
├── controllers/      # Sails Actions2 controllers, organized by feature
├── models/           # Waterline ORM models (User, Subscription, Quote, etc.)
├── helpers/          # Reusable logic, organized by domain (stripe/, salesforce/, ai/, etc.)
├── policies/         # Auth middleware (is-logged-in, is-super-admin, etc.)
├── responses/        # Custom response handlers (unauthorized, expired, badConfig)
└── hooks/custom/     # Server initialization, security headers, globals
assets/
├── js/
│   ├── components/   # Vue/Parasails components (*.component.js)
│   ├── pages/        # Page scripts (parasails.registerPage)
│   └── utilities/    # Shared utilities (parasails.registerUtility)
└── styles/           # LESS stylesheets
views/
├── layouts/          # EJS layout templates
├── pages/            # Page templates
├── partials/         # Reusable template fragments
└── emails/           # Email templates
config/
├── routes.js         # All route definitions
├── policies.js       # Route-to-policy mappings
├── custom.js         # App settings (API keys, TTLs, feature flags)
└── local.js          # Local overrides (not committed)
```

## Backend conventions

### Controllers & helpers
Both use the Sails Actions2 machine format (`friendlyName`, `inputs`, `exits`, `fn`). Call helpers with `await sails.helpers.domain.name.with({...})`. Throw exit names (e.g., `throw 'notFound'`) to trigger non-success exits.

### Models (Waterline ORM)
Declarative attribute schemas in `api/models/`. Use `protect: true` for sensitive fields (passwords, tokens).

### Routes
All in `config/routes.js`. Webhooks need `csrf: false`.

### Authentication
- Session-based: `req.session.userId`
- Logged-in user auto-hydrated as `req.me`
- Policies in `config/policies.js` control access; `'*': 'is-logged-in'` is the default

### Configuration
- `config/custom.js` — app settings, integration keys, feature flags
- `config/local.js` — local dev overrides (not committed, not deployed)
- `config/env/production.js` — production overrides
- Sensitive credentials go in `config/local.js` or environment variables, never in committed config

## Frontend conventions

### Data flow from controllers to pages
Values returned by a page's view action (e.g., `api/controllers/view-pricing.js`) are sent to the page in the `data` object. In page scripts, they're available on `this` (e.g., `this.pricingTable`). In templates:
- **EJS** (`<%- pricingTable %>`) — for server-side rendering of data from the view action
- **Vue** (`{{pricingTable}}`) — for values that change based on user interaction (filters, toggles, etc.)

Use EJS when the data is static from the server. Use Vue templates when the value is reactive and updated by page methods.

### Reusable components
Several Parasails components are used across multiple pages:
- `<scrollable-tweets>` — testimonial carousel. Requires `testimonialsForScrollableTweets` data from the view action (see Testimonials below).
- `<parallax-city>` — animated city skyline banner, used at the bottom of landing pages. Must sit at the top level of the page, outside `page-container`/`page-content`, so it can span the full viewport width with no padding. Typically appears just after a `bottom-gradient` section. See `views/pages/landing-pages/linux-management.ejs` for the full end-of-page structure.
- `<logo-carousel>` — rotating customer logo strip, typically placed in hero sections.
- `<modal>` — modal dialog. Control visibility with `v-if="modal === 'modal-name'"` and `@close="closeModal()"`. Commonly used for video embeds.

#### Video modal pattern
Landing pages typically include a "See Fleet in action" video button. The pattern requires:
1. Page script: add `modal: ''` to `data`, plus `clickOpenVideoModal` and `closeModal` methods
2. Template: add a `<modal purpose="video-modal">` with a YouTube iframe
3. LESS: include responsive video modal styles (see `assets/styles/pages/landing-pages/linux-management.less` for reference)

#### Testimonials
Testimonials are defined in `handbook/company/testimonials.yml` and compiled into `sails.config.builtStaticContent.testimonials`. Each has `quote`, `quoteAuthorName`, `quoteAuthorJobTitle`, `productCategories` (e.g., `Device management`, `Observability`, `Software management`), and optional media fields.

View actions that use `<scrollable-tweets>` must filter/sort testimonials and return them as `testimonialsForScrollableTweets`. See `api/controllers/landing-pages/view-linux-management.js` for the pattern.

### Cloud SDK (API calls)
Frontend-to-backend API calls use `Cloud.*` methods, invoked by the `ajax-form` component or via a page script's `handleSubmitting` function. Each Cloud method maps to a backend action. After adding or renaming an action, regenerate the SDK:
```bash
sails run rebuild-cloud-sdk
```

### Ajax forms
Form submission uses `<ajax-form>` — either `action="cloudMethodName"` for simple cases or `:handle-submitting="fn"` for custom logic. State props (`syncing`, `cloudError`, `formErrors`) use `.sync`. See `views/pages/contact.ejs` for a full example; see `assets/js/components/ajax-form.component.js` for supported validation rules.

### Global browser variables
`parasails`, `Cloud`, `io`, `_` (Lodash), `$` (jQuery), `moment`, `bowser`, `Vue`, `Stripe`, `gtag`, `ace`

### Image naming
Images in `assets/images/` follow the pattern: `{category}-{descriptor}-{css-dimensions}@2x.{extension}`

The dimensions in the filename are CSS pixels (half the actual pixel resolution). For example, a 32x32 pixel image used at 16x16 CSS pixels:
```
icon-checkmark-green-16x16@2x.png
```

## CSS/LESS conventions

### Preprocessor & build
LESS compiled via Grunt. Single entry point: `assets/styles/importer.less` imports everything. New `.less` files must be `@import`ed in `importer.less` to take effect.

### Selector convention
**Use `[purpose='name']` attribute selectors** — this is the primary styling approach, not traditional CSS classes:
```less
// In EJS template:
// <div purpose="hero-container">...</div>

// In LESS:
[purpose='hero-container'] {
  padding: 80px 0;
}
```
Nest `[purpose]` selectors to scope styles within a section. Traditional CSS classes are secondary — used only for Bootstrap utilities and state toggles (`.truncated`, `.expanded`, `.loading-spinner`).

### Page-level scoping
Each page stylesheet is scoped to a page ID selector at the root:
```less
#pricing {
  // All page-specific styles nested inside
  [purpose='page-content'] { ... }
  [purpose='hero-text'] { ... }

  @media (max-width: 991px) { ... }
}
```
This prevents style leakage between pages. The page ID matches the `id` attribute on the page's outermost `<div>` in the EJS template.

Some pages use a `-page` suffix (e.g., `#software-management-page` instead of `#software-management`). This is done when the base name would collide with an auto-generated heading ID — for example, markdown articles with a "Software management" heading get `id="software-management"` automatically. Add the `-page` suffix when the page name could conflict with a heading ID elsewhere on the site.

### Variables and mixins
All colors, fonts, weights, and mixins live in `mixins-and-variables/`. Always use variable names instead of raw hex (e.g., `@core-fleet-black` not `#192147`). Common mixins: `.page-container()`, `.page-content()`, `.btn-reset()`, `.fade-in()`.

Don't use `@core-vibrant-blue` in new code — it's deprecated.

Primary CTA buttons should use the `btn btn-primary` Bootstrap classes — this adds pseudo-element shine effects on hover (defined in `bootstrap-overrides.less`). The default color is `@core-vibrant-green` but can be overridden per page; the key benefit is the shine, not the color.

### Page backgrounds
Pages don't set their own section backgrounds. The page background is a gradient defined in `layout.less` and overridden per-page. Pages with a `<parallax-city>` footer typically end with a dedicated `bottom-gradient` section just before the component.

### Responsive breakpoints
Max-width media queries, typically nested inside the page's root ID selector:
```less
#my-page {
  // Desktop styles at root level

  @media (max-width: 1199px) { /* large desktop adjustments */ }
  @media (max-width: 991px)  { /* tablet: cards stack, padding reduces */ }
  @media (max-width: 767px)  { /* mobile: single column, smaller text */ }
  @media (max-width: 575px)  { /* small mobile: minimal padding */ }
  @media (max-width: 375px)  { /* extra small: final adjustments */ }
}
```

### Framework
Bootstrap 4 is loaded as a base dependency. Global overrides live in `bootstrap-overrides.less`, page-specific overrides should be scoped inside the page's ID selector.

Avoid using Bootstrap utility classes (`.d-flex`, `.justify-content-center`, `.flex-column`, etc.) for layout and display properties. Define these styles in the LESS stylesheet using `[purpose]` selectors instead — this keeps all styles in one place and makes them easier to adjust later. Bootstrap's grid (`.row`, `.col-*`) is acceptable where already established, but prefer stylesheet-defined layout for new work.

### Browser compatibility

The website enforces minimum browser versions via a [bowser](https://github.com/lancedikson/bowser) check in `views/layouts/layout.ejs` (around line 970). Visitors on unsupported browsers see a full-page block prompting them to upgrade. These floors were chosen to enable modern CSS features — notably the flexbox/grid `gap` property.

**Minimum supported versions** (source of truth: `layout.ejs`):

| Browser | Min version | Notes |
|---------|------------|-------|
| Chrome | 84 | `gap` support |
| Edge | 84 | `gap` support |
| Opera | 70 | `gap` support |
| Safari | 14 | `gap` support |
| Firefox | 103 | `backdrop-filter` support |
| iOS | 14 | Supports embedded podcast player |
| Android | 6 | Google's search crawler user agent |

Internet Explorer is blocked entirely.

**What's safe to use**:
- Flexbox and CSS Grid, including `gap` on both
- `backdrop-filter`
- CSS custom properties (variables) — supported everywhere above IE
- Modern ES2017+ JavaScript (async/await, object spread, etc.)

**What to be cautious with**:
- Container queries — Safari 14 does not support them; need to fall back to media queries or wait to raise the floor
- `:has()` selector — Safari 14 does not support it
- Any CSS feature newer than ~2021 — check [caniuse.com](https://caniuse.com) against the table above

**Manual QA**: Per the [handbook](https://fleetdm.com/handbook/engineering#check-browser-compatibility-for-fleetdm-com), cross-browser checks are done monthly via BrowserStack. Google Chrome (macOS) latest is the baseline; other supported browsers are checked against it. File issues as bugs and assign for fixing.

**Raising or lowering the floor**: Update the `LATEST_SUPPORTED_VERSION_BY_USER_AGENT` and `LATEST_SUPPORTED_VERSION_BY_OS` objects in `views/layouts/layout.ejs`. Add a comment explaining *why* that version was chosen (which CSS/JS feature it enables), matching the existing pattern.

### LESS formatting rules (from `.lesshintrc`)
- One space before `{` — `[purpose='hero'] {` not `[purpose='hero']{`
- One space after `:` in properties — `padding: 16px` not `padding:16px`
- Avoid `!important` — if unavoidable, add `//lesshint-disable-line importantRule` on the same line
- No strict property ordering enforced
- Zero warnings policy — `npm run lint` must pass with zero lesshint warnings

## Markdown content pipeline

### Source files
Markdown content lives outside the `website/` directory in three top-level folders:
- `docs/` — technical documentation
- `articles/` — blog posts, case studies, whitepapers, comparisons
- `handbook/` — internal company handbook

### Build process
The build script `scripts/build-static-content.js` compiles markdown to HTML:
```bash
sails run build-static-content    # compile markdown → EJS partials
npm run build-for-prod            # full production build (includes above + asset minification)
npm run start-dev                 # dev mode (runs build-static-content then starts console)
```
Compiled output lands in `views/partials/built-from-markdown/`; metadata is exposed at runtime as `sails.config.builtStaticContent`.

### Metadata
Embedded as HTML `<meta name="X" value="Y">` tags in the markdown file (not YAML frontmatter). See existing files in each folder for the required tags per content type.

### Custom syntax
- `((bubble-text))` — converts to `<bubble type="bubble-text">` elements
- Blockquotes — automatically rendered with `purpose="tip"` styling
- Code blocks — language-specific highlighting (`js`, `bash`, `yaml`, `mermaid`, etc.)
- Checklists — `- [x]` and `- [ ]` syntax renders as checkboxes

### Restrictions
The build script enforces several rules and will throw errors for:
- Vue template syntax (`{{ }}`) outside code blocks (conflicts with client-side Vue)
- Relative markdown links without `.md` extension
- `@fleetdm.com` email addresses in markdown
- Missing required meta tags per content type

## Creating new pages

**Always use `sails generate page <name>` or `sails generate page <folder>/<name>` — don't hand-create the controller/view/script/LESS files.** The generator produces the correct Actions2 shape, view path, and locals boilerplate; manual scaffolding is error-prone (wrong exits, missing `exposeLocalsToBrowser()` footer, etc.).

The generator creates four files. For root-level pages, use `<name>` paths; for nested pages, include the `<folder>/` segment:
- `api/controllers/view-<name>.js` or `api/controllers/<folder>/view-<name>.js`
- `views/pages/<name>.ejs` or `views/pages/<folder>/<name>.ejs`
- `assets/js/pages/<name>.page.js` or `assets/js/pages/<folder>/<name>.page.js`
- `assets/styles/pages/<name>.less` or `assets/styles/pages/<folder>/<name>.less`

### After running the generator
1. Add the route in `config/routes.js` with `pageTitleForMeta` and `pageDescriptionForMeta` under `locals`.
2. Add the matching import to `assets/styles/importer.less`: `@import 'pages/<name>.less';` for root-level pages or `@import 'pages/<folder>/<name>.less';` for nested pages.
3. If the page needs to bypass `is-logged-in`, update `config/policies.js` (not needed under folders already bypassing it, e.g. `landing-pages/`).
4. Re-lift the dev server — backend changes don't hot-reload.

### Reuse existing styles, layout, and elements
New pages should mirror the structure and styling of existing landing pages rather than inventing new patterns. Before writing markup or LESS, open 1–2 existing landing pages in `views/pages/landing-pages/` (e.g. `linux-management.ejs`, `replace-jamf.ejs`) and their paired stylesheets in `assets/styles/pages/landing-pages/`, and copy the section scaffolding — hero, feature rows, `<scrollable-tweets>`, `bottom-gradient`, `<parallax-city>` footer — along with the `[purpose='...']` naming conventions. Reuse existing components (`<logo-carousel>`, `<animated-arrow-button>`, `<modal>`, video modal pattern) instead of building one-off equivalents. Only introduce new `[purpose]` blocks or LESS variables when nothing existing fits.

### Deprecated
Do **not** use `sails generate landing-page` (the custom generator under `website/generators/landing-page/`). It's deprecated; use `sails generate page` for landing pages too.

## Code style

- **Indentation**: 2 spaces
- **Quotes**: Single quotes (template literals allowed)
- **Semicolons**: Always required
- **Equality**: Strict only (`===` / `!==`)
- **Variables/functions**: camelCase
- **Files/directories**: kebab-case

## Development commands

```bash
npm run start-dev       # Start dev server with live reload
npm run lint            # Run ESLint + HTMLHint + lesshint
npm run build-for-prod  # Compile markdown, build and minify assets
```

## Linting

- **JS**: ESLint (`.eslintrc` at root, browser override in `assets/.eslintrc`)
- **HTML/EJS**: HTMLHint (`.htmlhintrc`)
- **LESS**: lesshint (`.lesshintrc`) — zero warnings policy

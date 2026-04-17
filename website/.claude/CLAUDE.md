# CLAUDE.md — Fleet Website

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

### Controllers (Actions2 format)
Every controller uses the Sails machine pattern:
```javascript
module.exports = {
  friendlyName: 'View pricing',
  description: 'Display the pricing page.',
  exits: {
    success: { viewTemplatePath: 'pages/pricing' },
  },
  fn: async function () {
    return {};
  }
};
```
- File naming: `kebab-case.js` matching the action name in `config/routes.js`
- Always use `async/await` in `fn`
- Throw exit names to trigger non-success exits: `throw 'notFound';`
- Use `.intercept()` for Waterline error mapping

### Helpers
Same Actions2 machine format as controllers. Located in `api/helpers/`, organized by domain:
```javascript
module.exports = {
  friendlyName: 'Do something',
  inputs: { /* typed, validated inputs */ },
  fn: async function (inputs) { /* ... */ }
};
```
Call with `await sails.helpers.domainName.helperName.with({...})`.

### Models (Waterline ORM)
Declarative attribute schemas in `api/models/`. Use `protect: true` for sensitive fields (passwords, tokens).

### Routes
All defined in `config/routes.js`. Pattern: `'METHOD /path': { action: 'folder/action-name' }`.
- View routes use `locals:` for page metadata (title, description)
- Webhooks disable CSRF: `csrf: false`

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

### Templates (EJS)
- Pages: `views/pages/<name>.ejs`
- Partials: `views/partials/<name>.partial.ejs`
- Layouts: `views/layouts/layout.ejs`
- Use `<%- partial('path') %>` for includes, `<%= var %>` for escaped output

### Data flow from controllers to pages
Values returned by a page's view action (e.g., `api/controllers/view-pricing.js`) are sent to the page in the `data` object. In page scripts, they're available on `this` (e.g., `this.pricingTable`). In templates:
- **EJS** (`<%- pricingTable %>`) — for server-side rendering of data from the view action
- **Vue** (`{{pricingTable}}`) — for values that change based on user interaction (filters, toggles, etc.)

Use EJS when the data is static from the server. Use Vue templates when the value is reactive and updated by page methods.

### Page scripts (Parasails)
Each page has a corresponding script in `assets/js/pages/`:
```javascript
parasails.registerPage('pricing', {
  data: { /* reactive data */ },
  beforeMount: function () { /* ... */ },
  mounted: async function () { /* ... */ },
  methods: { /* ... */ }
});
```

### Components (Vue/Parasails)
Reusable UI components in `assets/js/components/`:
```javascript
parasails.registerComponent('ajaxButton', {
  props: ['syncing'],
  template: '#ajax-button',
  data: function () { return {}; },
  methods: { /* ... */ }
});
```
- File naming: `kebab-case.component.js`
- Template markup lives in the EJS partial, referenced by `template: '#component-id'`

### Utilities
Shared functions in `assets/js/utilities/`:
```javascript
parasails.registerUtility('openStripeCheckout', async function openStripeCheckout() { /* ... */ });
```

### Reusable components
Several Parasails components are used across multiple pages:
- `<scrollable-tweets>` — testimonial carousel. Requires `testimonialsForScrollableTweets` data from the view action (see Testimonials below).
- `<parallax-city>` — animated city skyline banner, used at the bottom of landing pages.
- `<logo-carousel>` — rotating customer logo strip, typically placed in hero sections.
- `<modal>` — modal dialog. Control visibility with `v-if="modal === 'modal-name'"` and `@close="closeModal()"`. Commonly used for video embeds.

#### Video modal pattern
Landing pages typically include a "See Fleet in action" video button. The pattern requires:
1. Page script: add `modal: ''` to `data`, plus `clickOpenVideoModal` and `closeModal` methods
2. Template: add a `<modal purpose="video-modal">` with a YouTube iframe
3. LESS: include responsive video modal styles (see `assets/styles/pages/landing-pages/linux-management.less` for reference)

#### Testimonials
Testimonials are defined in `handbook/company/testimonials.yml` and compiled into `sails.config.builtStaticContent.testimonials` by `scripts/build-static-content.js`. Each testimonial has: `quote`, `quoteAuthorName`, `quoteAuthorJobTitle`, `quoteAuthorProfileImageFilename`, `quoteLinkUrl`, `productCategories` (array of `Observability`, `Device management`, `Software management`), and optionally `quoteImageFilename` and `youtubeVideoUrl`.

The view action must filter and sort testimonials for the page's context, then return them to the template:
```javascript
// In the view action's fn:
let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);
let testimonialOrderForThisPage = [
  'Author Name 1',
  'Author Name 2',
  // ...
];
testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial) => {
  return _.contains(testimonial.productCategories, 'Device management') &&
         _.contains(testimonialOrderForThisPage, testimonial.quoteAuthorName);
});
testimonialsForScrollableTweets.sort((a, b) => {
  return testimonialOrderForThisPage.indexOf(a.quoteAuthorName) -
         testimonialOrderForThisPage.indexOf(b.quoteAuthorName);
});
return { testimonialsForScrollableTweets };
```
Reference: `api/controllers/landing-pages/view-linux-management.js`

### Cloud SDK (API calls)
Frontend-to-backend API calls use `Cloud.*` methods, invoked by the `ajax-form` component or via a page script's `handleSubmitting` function. Each Cloud method maps to a backend action. After adding or renaming an action, regenerate the SDK:
```bash
sails run rebuild-cloud-sdk
```

### Ajax forms
The `<ajax-form>` component handles form submission, client-side validation, loading state, and error display. It either calls a Cloud SDK method directly or delegates to a custom handler.

#### Props
| Prop | Type | Description |
|------|------|-------------|
| `action` | string | Name of a `Cloud.*` method to call (e.g., `action="login"` → `Cloud.login()`). Mutually exclusive with `handleSubmitting`. |
| `handleSubmitting` | async function | Custom submission handler. Receives parsed form data (`argins`), must call Cloud methods manually. Use for complex logic (multi-step flows, conditional redirects, response inspection). |
| `formData` | object | Reactive object bound to form inputs via `v-model`. Keys are field names, values are input values. |
| `formRules` | object | Client-side validation rules keyed by field name. Only works with `formData`. |
| `formErrors` | object | Tracks validation errors. Bound via `.sync`. Keys are field names, values are the rule that failed. |
| `syncing` | boolean | Bound via `.sync`. `true` while an AJAX request is in-flight. Prevents double-posting. |
| `cloudError` | string | Bound via `.sync`. Captures server-side exit signals (e.g., `'emailAlreadyInUse'`). Cleared before each submission. |
| `handleParsing` | async function | Custom function to transform form data before submission. Return `undefined` to abort. |

All state props (`syncing`, `cloudError`, `formErrors`) use Vue's `.sync` modifier:
```ejs
<ajax-form :syncing.sync="syncing" :cloud-error.sync="cloudError" :form-errors.sync="formErrors">
```

#### Events
- `@submitted` — fires after a successful response. Receives result data from the server.
- `@rejected` — fires after a failed response. Receives the error object (rarely used; `cloudError` is preferred).

#### Validation rules
```javascript
formRules: {
  emailAddress: { required: true, isEmail: true },
  password: { required: true, minLength: 8 },
  confirmPassword: { required: true, sameAs: 'password' },
  agreeToTerms: { is: true },
  role: { isIn: ['admin', 'user', 'observer'] },
  custom: (value) => /^[A-Z]/.test(value),  // must return truthy/falsy
}
```
Optional fields skip all rules when empty — only `required: true` checks for empty values.

#### Pattern 1: Simple form with `action`
Use when the form maps directly to a single Cloud method with no special handling:

**Template:**
```ejs
<ajax-form action="deliverContactFormMessage"
  :form-data="formData"
  :form-rules="formRules"
  :form-errors.sync="formErrors"
  :syncing.sync="syncing"
  :cloud-error.sync="cloudError"
  @submitted="submittedForm()">
  <div class="form-group">
    <label for="email">Work email *</label>
    <input class="form-control" type="email"
      :class="[formErrors.emailAddress ? 'is-invalid' : '']"
      v-model.trim="formData.emailAddress"
      autocomplete="email" focus-first>
    <div class="invalid-feedback" v-if="formErrors.emailAddress">Please enter a valid email.</div>
  </div>
  <cloud-error v-if="cloudError === 'invalidEmailDomain'">Please use a work email.</cloud-error>
  <cloud-error v-else-if="cloudError"></cloud-error>
  <ajax-button type="submit" :syncing="syncing" class="btn btn-primary">Send</ajax-button>
</ajax-form>
```

**Page script:**
```javascript
data: {
  formData: { /* empty or with defaults */ },
  formErrors: {},
  formRules: {
    emailAddress: { required: true, isEmail: true },
    firstName: { required: true },
  },
  syncing: false,
  cloudError: '',
},
methods: {
  submittedForm: async function() {
    // Handle success — show message, redirect, fire analytics, etc.
    this.cloudSuccess = true;
  },
}
```

#### Pattern 2: Custom handler with `handleSubmitting`
Use when you need to inspect the response, call multiple Cloud methods, or do conditional logic:

**Template:**
```ejs
<ajax-form :handle-submitting="handleSubmittingDemoForm"
  :form-data="formData"
  :form-rules="formRules"
  :form-errors.sync="formErrors"
  :syncing.sync="syncing"
  :cloud-error.sync="cloudError">
  <!-- form inputs -->
  <ajax-button type="submit" :syncing="syncing" class="btn btn-primary">Submit</ajax-button>
</ajax-form>
```

**Page script:**
```javascript
handleSubmittingDemoForm: async function(argins) {
  let result = await Cloud.deliverDemoFormSubmission.with(argins);
  // Inspect response, fire analytics, redirect conditionally
  if (result.qualified) {
    this.goto(result.calendlyUrl);
  } else {
    this.goto('/thanks');
  }
},
```

Note: when using `handleSubmitting`, you don't need `@submitted` — handle everything in the function itself. The component still manages `syncing` and `cloudError` automatically.

#### Error display
```ejs
<!-- Specific error messages for known exit codes -->
<cloud-error v-if="cloudError === 'emailAlreadyInUse'">This email is already registered.</cloud-error>
<!-- Generic fallback for unexpected errors -->
<cloud-error v-else-if="cloudError"></cloud-error>
```

#### Focus management
Add `focus-first` to the first input — the component auto-focuses it on non-mobile browsers:
```ejs
<input v-model.trim="formData.emailAddress" focus-first>
```

Reference: `views/pages/contact.ejs`, `views/pages/entrance/login.ejs`

### Global browser variables
`parasails`, `Cloud`, `io`, `_` (Lodash), `$` (jQuery), `moment`, `bowser`, `Vue`, `Stripe`, `gtag`, `ace`

### File naming
kebab-case everywhere: `view-pricing.js`, `ajax-button.component.js`, `pricing.ejs`, `homepage.less`

### Image naming
Images in `assets/images/` follow the pattern: `{category}-{descriptor}-{css-dimensions}@2x.{extension}`

The dimensions in the filename are CSS pixels (half the actual pixel resolution). For example, a 32x32 pixel image used at 16x16 CSS pixels:
```
icon-checkmark-green-16x16@2x.png
```

## CSS/LESS conventions

### Preprocessor & build
LESS compiled via Grunt. Single entry point: `assets/styles/importer.less` imports everything. New stylesheets must be `@import`ed in `importer.less` to take effect.

### File organization
Import order in `importer.less`:
1. `mixins-and-variables/` — variables and mixins only, no selectors (colors, typography, buttons, animations, containers, truncate)
2. `bootstrap-overrides.less` — Bootstrap 4 customizations
3. `layout.less` — global layout (header, footer, page-wrap)
4. `components/*.component.less` — per-component styles
5. `pages/*.less` — per-page styles (one file per page, can be nested in subdirectories like `pages/entrance/`, `pages/docs/`)

When adding a new page or component, create a corresponding `.less` file and add its `@import` to `importer.less` in the correct section.

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
- **Page elements**: `[purpose='page-container']`, `[purpose='page-content']`, `[purpose='hero-text']`, `[purpose='button-row']`, `[purpose='cta-button']`, etc.
- **Parasails components**: `[parasails-component='modal']`, `[parasails-component='animated-arrow-button']`
- **Nesting**: Nest `[purpose]` selectors to scope styles within a section:
```less
[purpose='faq'] {
  [purpose='accordion'] {
    [purpose='accordion-header'] {
      font-weight: 700;
    }
  }
}
```
- Traditional CSS classes are secondary — used for Bootstrap utilities (`.btn`, `.d-flex`), state toggles (`.truncated`, `.expanded`, `.collapsed`), and loading indicators (`.loading-spinner`).

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

### Naming
- `purpose` attribute values: kebab-case (`purpose='hero-container'`, `purpose='cta-button'`, `purpose='pricing-card-body'`)
- CSS class names: kebab-case (`.loading-spinner`, `.loading-dot`)
- LESS variables: kebab-case with `@` prefix (`@core-fleet-black`, `@page-padding`)
- LESS mixins: kebab-case as function calls (`.btn-reset()`, `.page-container()`)

### Variables

#### Colors (`mixins-and-variables/colors.less`)
```less
// Core palette
@core-fleet-black: #192147;
@core-fleet-black-75: #515774;     // secondary text
@core-fleet-black-50: #8b8fa2;
@core-fleet-black-33: #B3B6C1;
@core-fleet-black-25: #C5C7D1;
@core-fleet-black-10: #E2E4EA;     // borders

@core-vibrant-red: #FF5C83;        // CTAs, accents
@core-vibrant-green: #009A7D;      // buttons, positive
@core-vibrant-blue: #6A67FE;       // links, highlights
@brand: #14acc2;                   // brand teal

// Backgrounds & UI
@ui-off-white: #F9FAFC;            // alternating rows, subtle bg
@ui-gray: #E3E3E3;
@error: @core-vibrant-red;

// Text
@text-normal: @core-fleet-black;
@text-muted: lighten(@text-normal, 60%);
```
Always use variable names instead of raw hex values when a matching variable exists. For example, use `@core-fleet-black` not `#192147`, and `@core-fleet-black-75` not `#515774`.

Primary CTA buttons should use the `btn btn-primary` Bootstrap classes. This adds pseudo-element shine effects on hover (defined in `bootstrap-overrides.less`). The default color is `@core-vibrant-green`, but can be overridden per page — the key benefit of the class is the shine effect, not the color.

#### Typography (`mixins-and-variables/typography.less`)
```less
@main-font: 'Inter', sans-serif;
@header-font: 'Inter', sans-serif;
@code-font: 'Source Code Pro', sans-serif;
@bold: 700;
@normal: 400;
```

### Mixins

#### Container mixins (`mixins-and-variables/containers.less`)
Use these for consistent page layout instead of writing raw padding/max-width:
```less
// For [purpose='page-container']:
.page-container()        // Standard pages: 64px padding, responsive step-downs
.page-container-docs()   // Docs pages: 48px padding, responsive step-downs

// For [purpose='page-content']:
.page-content()          // Standard: 1072px max-width, centered
.page-content-docs()     // Docs: 1104px max-width, centered
.page-content-narrow()   // Forms: 528px max-width, centered

// For smaller centered containers:
.container-sm()          // 450px max-width
.container-md()          // 650px max-width

// Wide page container with built-in max-width:
.page-container-wide()   // 1200px max-width + 64px padding, centered
```

#### Button mixins (`mixins-and-variables/buttons.less`)
```less
.btn-reset()                // Strip all browser button defaults
.btn-animated-arrow-red()   // Red arrow CTA with hover animation
.btn-animated-arrow-white() // White arrow CTA with hover animation
```

#### Animation mixins (`mixins-and-variables/animations.less`)
```less
.fade-in()              // Opacity 0→1 keyframe animation
.loader(@dot-color)     // Loading dots animation
.bob()                  // Gentle vertical float animation
.skid()                 // Horizontal shake animation
.fly-fade()             // Horizontal fly with fade in/out
.transition(@transition)  // Cross-browser transition shorthand
.translate(@x, @y:0)     // Cross-browser transform translate
.animation-delay(@delay)  // Cross-browser animation-delay
.animation-duration(@d)   // Cross-browser animation-duration
```

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
Group all responsive overrides for a page inside the page's scope, typically at the bottom of the file.

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
- `articles/` — blog posts, case studies, whitepapers
- `handbook/` — internal company handbook

### Build process
The build script `scripts/build-static-content.js` compiles markdown to HTML:
```bash
sails run build-static-content    # compile markdown → EJS partials
npm run build-for-prod            # full production build (includes above + asset minification)
npm run start-dev                 # dev mode (runs build-static-content then starts console)
```

Markdown is converted to HTML using `marked` via the `sails.helpers.strings.toHtml()` helper. Compiled output is written as EJS partials to `views/partials/built-from-markdown/` with filenames like `articles--my-post-title--b99a63c246.ejs`. Page metadata is stored in `.sailsrc` under `builtStaticContent` and accessed at runtime via `sails.config.builtStaticContent`.

### Metadata
Metadata is embedded as HTML `<meta>` tags in the markdown files (not YAML frontmatter):
```markdown
<meta name="articleTitle" value="My Article Title">
<meta name="authorFullName" value="Jane Smith">
<meta name="authorGitHubUsername" value="janesmith">
<meta name="category" value="security">
<meta name="description" value="A short description for SEO.">
```

Common meta tags by content type:
- **Docs**: `pageOrderInSection` (sort order)
- **Articles**: `articleTitle`, `authorFullName`, `authorGitHubUsername`, `category`, `description`, `articleImageUrl`
- **Handbook**: `maintainedBy` (GitHub username)

### Heading IDs
Headings automatically get kebab-case IDs (e.g., `## Global activity webhook` → `id="global-activity-webhook"`). This is why some page stylesheets use a `-page` suffix on their ID selector — to avoid collisions with auto-generated heading IDs.

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

Use the built-in Sails page generator:
```bash
sails generate page <kebab-cased-page-name>
# Or with a subfolder:
sails generate page <folder>/<kebab-cased-page-name>
# Example:
sails generate page landing-pages/my-new-page
```

This generates four files: controller, view template, page script, and LESS stylesheet. After generating, you still need to:
1. **Add the route** in `config/routes.js` with page metadata in `locals`:
   ```javascript
   'GET /pricing': {
     action: 'view-pricing',
     locals: {
       pageTitleForMeta: 'Pricing',
       pageDescriptionForMeta: 'Use Fleet for free or get started with Fleet Premium.',
     },
   },
   ```
2. **Add the LESS import** to `assets/styles/importer.less` in the per-page section
3. **Update policies** in `config/policies.js` if the page needs to bypass the default `is-logged-in` policy (not needed if the page lives under a folder that already bypasses it, like `landing-pages/`)

## Creating a new whitepaper

Whitepapers are a special article type with a gated download form. They use a dedicated template (`pages/articles/basic-whitepaper`), a separate route prefix (`/whitepapers/`), and require a PDF file.

### Step-by-step

1. **Create the markdown file** in the top-level `articles/` directory (outside `website/`):
   ```
   articles/my-whitepaper-title.md
   ```

2. **Add required meta tags** at the top of the markdown:
   ```html
   <meta name="articleTitle" value="My Whitepaper Title">
   <meta name="authorFullName" value="Author Name">
   <meta name="authorGitHubUsername" value="github-username">
   <meta name="category" value="whitepaper">
   <meta name="publishedOn" value="2026-04-13">
   <meta name="description" value="A short description for SEO (max 150 chars).">
   <meta name="articleImageUrl" value="../website/assets/images/articles/my-whitepaper-cover-504x336@2x.png">
   <meta name="whitepaperFilename" value="my-whitepaper-file.pdf">
   <meta name="introductionTextBlockOne" value="First paragraph shown above the fold.">
   <meta name="introductionTextBlockTwo" value="Second paragraph shown above the fold.">
   ```
   - `category` must be `whitepaper`
   - `whitepaperFilename` is required — the build will fail without it
   - `introductionTextBlockOne` and `introductionTextBlockTwo` are displayed as preview text above the download form
   - `articleImageUrl` should follow the image naming convention (`{descriptor}-{css-dimensions}@2x.{ext}`)

3. **Place the PDF** in `website/assets/pdfs/`. The filename must match the `whitepaperFilename` meta tag exactly. The build script verifies this file exists.

4. **Place the cover image** in `website/assets/images/articles/`. The path must match `articleImageUrl`.

5. **Write the markdown content** below the meta tags. This is the full whitepaper content that appears on the page after the user downloads.

6. **Build and test**:
   ```bash
   sails run build-static-content   # Compiles markdown, validates meta tags
   npm run start-dev                # Start dev server
   ```
   The whitepaper will be available at `/whitepapers/{slug}` (slug is derived from the markdown filename).

### How it works
- Routes are defined in `config/routes.js`: `'GET /whitepapers/:slug'` maps to `api/controllers/articles/view-basic-whitepaper.js`
- The controller looks up the page in `sails.config.builtStaticContent.markdownPages` by slug
- The template (`views/pages/articles/basic-whitepaper.ejs`) renders the intro text, a download form (first name, last name, work email), and the compiled article content
- On form submission, `api/controllers/deliver-whitepaper-download-request.js` validates the email, creates/updates Salesforce records, and the client triggers the PDF download
- No route or controller changes are needed when adding a new whitepaper — the slug-based route handles it automatically

## Creating a comparison article

Comparison articles use the `basic-comparison` template with a slug-based route (`/compare/{slug}`). They feature a sidebar "On this page" nav built from `h2` headings, and tables are enhanced with `data-label` attributes for mobile responsiveness.

### Step-by-step

1. **Create the markdown file** in the top-level `articles/` directory (outside `website/`):
   ```
   articles/fleet-vs-competitor-a-vs-competitor-b-comparison.md
   ```

2. **Write the article content** using these heading conventions:
   - `## Heading` (h2) for main sections — these appear in the sidebar nav
   - `### Subheading` (h3) for subsections within a main section
   - `#### Question text` (h4) for FAQ questions — do **not** use `###` for FAQs

3. **Add required meta tags** at the bottom of the markdown (this is where articles place their meta tags):
   ```html
   <meta name="articleTitle" value="Fleet vs. Competitor A vs. Competitor B">
   <meta name="authorFullName" value="Author Name">
   <meta name="authorGitHubUsername" value="github-username">
   <meta name="category" value="comparison">
   <meta name="articleSlugInCategory" value="competitor-a-vs-competitor-b-vs-fleet">
   <meta name="introductionTextBlockOne" value="First paragraph displayed above the fold.">
   <meta name="publishedOn" value="2026-04-14">
   <meta name="description" value="Short SEO description (max 150 chars).">
   ```
   - `category` must be `comparison`
   - `articleSlugInCategory` is required — this becomes the URL slug (`/compare/{slug}`)
   - `introductionTextBlockOne` is displayed as intro text above the article content
   - `introductionTextBlockTwo` is optional — a second intro paragraph

4. **Comparison tables** use standard Markdown pipe syntax:
   ```markdown
   | Dimension | Fleet | Competitor A | Competitor B |
   | --- | --- | --- | --- |
   | Feature X | Yes | Limited | No |
   ```
   The page script automatically reads table headers and adds `data-label` attributes to each cell for mobile-responsive display.

5. **Build and test**:
   ```bash
   sails run build-static-content   # Compiles markdown, validates meta tags
   npm run start-dev                # Start dev server
   ```
   The article will be available at `/compare/{articleSlugInCategory}`.

### Converting an existing article to a comparison

If an article was created with `category: articles` but should be a comparison:
1. Change `category` to `comparison`
2. Add `articleSlugInCategory` with the desired URL slug
3. Add `introductionTextBlockOne` (pulled from the article's opening paragraph)
4. Remove any standalone intro paragraph from the body (it's now in the meta tag)
5. Change any FAQ headings from `###` to `####`
6. The `articleTitle` meta tag becomes the page's h1 — remove any h1 from the markdown body

### How it works
- Route: `'GET /compare/:slug'` maps to `api/controllers/articles/view-basic-comparison.js`
- The controller looks up the page in `sails.config.builtStaticContent.markdownPages` by matching `/compare/{slug}`
- The template renders intro text, sidebar nav (from h2 headings), and compiled article content
- The page script (`basic-comparison.page.js`) extracts h2 headings for sidebar navigation and adds `data-label` attributes to table cells for mobile responsiveness
- No route or controller changes are needed when adding a new comparison — the slug-based route handles it automatically

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

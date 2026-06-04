# Frontend patterns

Cross-cutting expectations that aren't captured by component conventions alone. If you're adding a feature, scan this for anything that applies before opening the PR.

## Command palette

Every user-visible action in the product belongs in the command palette. It's the keyboard surface for power users — a feature that isn't there might as well not exist for them.

Source: `frontend/components/CommandPalette/`. Items are defined per group under `groups/`.

### When to add a palette entry

| Adding... | Goes in |
|---|---|
| A new top-level page (routed under a top nav item) | `groups/pages.ts` |
| A new write action (modal, form, create/edit page) | `groups/commands.ts` |
| A new view-by-search flow (like "View host") | `groups/commands.ts` with `opensSubPage: true`, plus a picker in `components/` |
| A new MDM platform or connector | `groups/mdm.ts` |
| A new automation hook | `groups/automations.ts` |
| A new settings page or admin sub-page | `groups/settings.ts` |
| A new control / policy / script feature | `groups/controls.ts` |
| A new software action or view | `groups/software.ts` |

Sub-pages of an existing palette item live in that item's `subItems` array, not as top-level entries. The user gets the sub-item when they expand the parent (chevron) or when their search promotes the sub-item into Best match.

### Required and optional fields

```ts
interface ICommandItem {
  id: string;                // unique kebab-case
  label: string;             // sentence case, verb first ("Add report")
  group: typeof GROUPS[number];
  path?: string;             // navigation target (use withTeamId() if team-scoped)
  onAction?: () => void;     // alternative to path for custom side effects
  keywords?: string[];       // synonyms + aliases — see below
  teamName?: string;         // chip shown when the action switches the user's fleet context
  subItems?: ICommandSubItem[];
  opensSubPage?: boolean;    // shows the chevron-right; required for picker actions
}
```

### Label conventions

- Sentence case: "Add report", not "Add Report".
- Verb first for actions: "Add", "Edit", "Delete", "Run", "View", "Manage", "Turn on" / "Turn off".
- No trailing punctuation.
- Match the destination page's own primary-button text where possible.
- Use **fleet** / **report** (current product terminology), not **team** / **query**.

### Keyword authoring

Best match scoring is **label-first**: any label match (exact, prefix, word-prefix, or substring) outranks any keyword match. This shapes how to write keywords.

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
- Repeat words from the label. `Add user` already covers searches for "add" or "user" via label tiers; adding `add user` as a keyword does nothing.
- Use multi-word keyword phrases when a single word works. Phrases only match when the full phrase is typed; single words match prefix and word-prefix automatically.
- Pile in low-signal substrings ("the", "some", generic verbs).

### Permission gating

Mirror the destination page's gate exactly. If the page rejects technicians, gate the palette item on `!isTechnician`. If the destination renders `<PremiumFeatureMessage />` on free tier, gate the palette item on `isPremiumTier`. The palette must not route users to a screen they can't use.

Use the existing context flags from `ICommandPaletteContext` (`canWrite`, `canAccessSettings`, `canAddSoftware`, `canManageReportAutomations`, etc.) and add new ones to `ICommandPaletteContext` + `CommandPalette.tsx` when an existing one doesn't model the destination's check.

### Team context (`teamName`)

Set `teamName` when invoking the action will switch the user's current fleet context. The palette renders it as a chip on the right so the user sees the upcoming switch before they click. Use the derived helpers from `groups/derivations.ts`:

- `switchesFromUnassigned` — destination requires a specific fleet, action invokable from Unassigned
- `switchesFromAllFleets` — destination requires a specific fleet, action invokable from All fleets
- `defaultDestination` — destination always lands on the default (e.g., "All fleets")

Don't hardcode team names; the helpers know which switches actually happen and return `undefined` when no chip is needed.

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

The scoring/ranking helpers (`scoreMatch`, `computeBestMatch`, `highlightMatches`) are already covered; you don't need to re-test the framework when adding an item.

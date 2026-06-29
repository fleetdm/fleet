---
name: command-palette
description: Authoring guide for the Fleet command palette. Use when adding or editing items in frontend/components/CommandPalette/groups/, when editing frontend/router/paths.ts or frontend/router/index.tsx, or when adding a new top-level page, global create action, MDM connector / singleton config, automation hook, or picker action that needs a palette entry.
allowed-tools: Read, Grep, Glob, Bash(yarn test*)
effort: medium
---

# Command palette authoring

The canonical guide lives in **`frontend/docs/patterns.md` § Command palette** — read that first. It covers what belongs (and what doesn't), the group-to-file mapping, required/optional fields, label conventions, the full keyword/synonym checklist, permission gating, `teamName` chips, search-only items, and test expectations.

This skill exists to make sure that guide gets followed when palette-worthy changes land.

## Before adding an item

1. **Read `frontend/docs/patterns.md` § Command palette** end to end.
2. **Grep the target group file** for similar existing items. Match their shape — field order, keyword style, gating, `teamName` helper usage — instead of inventing a new pattern. The groups are the source of truth for current conventions:
   ```
   frontend/components/CommandPalette/groups/
   ```
3. **Confirm the destination page's own permission check**, then mirror it on the palette item using a flag from `ICommandPaletteContext` (`frontend/components/CommandPalette/helpers.ts`). Add a new flag there only if no existing one models the destination's check. Don't route users to a screen they can't use.
4. **Premium paywall check:** if the destination page (or the specific tab/section the link lands on) renders `<PremiumFeatureMessage />` for `!isPremiumTier`, gate the palette item on `isPremiumTier` so it's hidden on Free. A palette entry that lands on the upsell wall is a bait-and-switch — the palette is for actions, not for marketing the paid tier. Mirror tab-level paywalls too (e.g., `paths.ADMIN_INTEGRATIONS_SSO_END_USERS` lands on a Premium-only tab even though the parent SSO page works on Free). If the destination only paywalls a sub-section (not the whole page or the linked tab), don't gate — the page is still useful on Free.

## After adding an item

1. **Update `frontend/components/CommandPalette/helpers.tests.ts`**:
   - Assert the item appears for the right roles and hides for the wrong ones
   - If premium-only: assert absence in the `Fleet Free (isPremiumTier: false)` block
   - If hidden in primo mode: add to the `Primo Mode (isPrimoMode: true)` block
   - If it sets a `teamName` chip: assert it renders / doesn't render against the relevant fleet contexts
2. Run the palette tests:
   ```
   yarn test frontend/components/CommandPalette/helpers.tests.ts
   ```
   (Note: test files use `.tests.ts` plural, and `yarn test` — not `yarn jest` — uses the project's jest config.)

## When *not* to add an entry

- Per-entity edit / delete operations (the entity is already in scope on its row or detail page)
- Bulk-select operations that depend on an existing selection
- One-off UI affordances (toggles, expanders) tied to a single view

See patterns.md for the dividing line and the full "doesn't belong" list.

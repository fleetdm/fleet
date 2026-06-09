---
name: command-palette
description: Authoring guide for the Fleet command palette. Use when adding or editing items in frontend/components/CommandPalette/groups/, when editing frontend/router/paths.ts or frontend/router/index.tsx, or when adding a new top-level page, global create action, MDM connector / singleton config, automation hook, or view-by-search flow that needs a palette entry.
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
3. **Confirm the destination page's permission gate**, then mirror it exactly on the palette item. Don't route users to a screen they can't use.

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

---
name: new-icon
description: Add or change a Fleet UI icon (frontend/components/icons, ICON_MAP) or a software/OS logo tile (frontend/pages/SoftwarePage/components/icons) so it follows Fleet's icon guidelines. Use when asked to add a new icon, redraw or replace an icon, convert an icon from Figma to code, or fix an icon's spacing, stroke, or alignment.
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(yarn test*), Bash(npx prettier*), Bash(npx eslint*), Bash(npx tsc*)
effort: medium
---

# Add or change a Fleet icon

Icon to add or change: $ARGUMENTS

The source of truth is the **Icons (new) - available from Fleet 5.0** page in the [🧩 Product design system](https://www.figma.com/design/gxvU745LfOdkE9AuRg64wi/?node-id=8901-7) Figma file. Its **Icon guidelines** spec has the full design rules. The code rules live in `frontend/docs/patterns.md` under "Icon rules". The "Icons (deprecated in Fleet 5)" page is the old set; don't use it.

## 0. Find out who you're helping

Ask the user, unless it's already clear:

1. **Is this a standalone icon task, or part of feature work?**
2. **Are they designing the icon or implementing it in code?**

Then follow the matching path. When a product designer and an engineer split the work, each one runs the skill for their part.

| | **Designing** | **Implementing** |
| --- | --- | --- |
| **Standalone task** | Steps 1–2, then add it to the design system page (step 2b) | Steps 1, 3–6 |
| **Feature work** | Steps 1–2 in the feature's Figma file, get approval (step 2a), then step 2b | Steps 1, 3–6, from the design system page only |

**Engineers implement icons only from the design system page.** If the link you're given points to a feature or story file instead, stop. Ask the designer to get the icon approved and added to the design system page first (step 2a and 2b), then continue with that component link. Don't ship an icon that isn't in the design system.

## 1. Check whether an icon already exists

- Search `ICON_MAP` in `frontend/components/icons/index.ts` and the design system Icons page for an icon with the same meaning. Reuse it instead of adding a near-duplicate or a variant.
- Use simple names: `error`, `pending`, `info`, `success`, `close`. Don't add `-alt`, `-outline`, or size suffixes for the same meaning.

## 2. Draw it in Figma

For feature work, draw the icon in the feature's Figma file. For a standalone task, draw it on a scratch page or straight in the design system file.

Every icon must follow these rules. If you're given an SVG that breaks them, fix it in Figma (or ask the designer) before adding it to code.

| **Icon** | **Frame** | **Stroke** |
| --- | --- | --- |
| UI icon (`icon/<name>`) | 16×16 | 1.5px |
| Large icon (`icon-32/<name>`) | 32×32 | 2px |
| `external-link` for 12px text only | 12×12 | 1px |

- **Size:** 16px is the minimum. Don't scale an icon to a new size; draw that size separately.
- **Strokes:** round caps and round joins on every stroke. Corner radius 1.5px on the stroke centerline. Keep all ink inside the frame.
- **Gaps inside an icon:** 1.5px between separate parts.
- **Gaps to a badge:** the outline ends 1.5px from the badge (2.5px on 32px icons). Keep every gap to the same badge equal.
- **Line ends:** where a line stops next to another shape, end it straight-on, not partway around a curve, so the gap reads the same as the others.
- **Outlined by default.** Use filled only when it's needed to convey meaning or importance (`error`, and `success-filled` for the OS settings "Verified" status). Badges are filled with the symbol cut out.
- **Color:** draw in `ui-fleet-black-75` (`#515774`), the default icon color. Other colors are applied in code.
- **Cut-outs** are real holes (even-odd fill), not white shapes.
- **Layers:** flat. Vectors sit directly in the icon frame, with no groups, masks, or boolean operations.

After editing paths in Figma, check again that caps and joins are still round. Replacing path data resets them.

If the Figma MCP server is connected, check the drawn icon against these rules with a read-only `use_figma` script: frame size, stroke weight, round caps and joins on every stroked vector, no groups or masks, all ink inside the frame, `#515774` paints only, and the `icon/<name>` name. Report anything that breaks a rule instead of fixing it silently.

### 2a. Get approval (new icons)

A new icon is a design system change. Follow the handbook's [design system request process](https://fleetdm.com/handbook/company/product-groups#design-system) to get approval from the Head of Design.

### 2b. Add it to the design system

After approval, add the icon as an `icon/<name>` (or `icon-32/<name>`) component on the **Icons (new) - available from Fleet 5.0** page. Then put that component's link (not the feature file) in the issue's dev notes, so the engineer implements from the design system.

## 3. Convert strokes to filled paths

Code icons use filled paths only. Several stylesheets recolor icons by setting `path { fill }` on hover or focus, so a stroked path would render as a solid shape.

- Start from the component on the design system Icons page. In Figma, select the icon's vectors and use **Outline stroke**, then copy the icon as SVG. With the Figma MCP server, export the component as SVG (`exportAsync({ format: "SVG_STRING" })`) and convert it locally instead.
- Without Figma, convert a stroked SVG with [picosvg](https://github.com/googlefonts/picosvg) (`pip install picosvg`, then `picosvg icon.svg`).
- Compare the result with the Figma design at large size before using it.

## 4. Add the UI icon component

Create `frontend/components/icons/<Name>.tsx`. Follow the existing icons (for example `Pending.tsx`):

```tsx
import React from "react";

import { COLORS, Colors } from "styles/var/colors";
import { ICON_SIZES, IconSizes } from "styles/var/icon_sizes";

interface IExampleProps {
  color?: Colors;
  size?: IconSizes;
}

const Example = ({
  color = "ui-fleet-black-75",
  size = "medium",
}: IExampleProps) => {
  return (
    <svg
      width={ICON_SIZES[size]}
      height={ICON_SIZES[size]}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="..."
        fill={COLORS[color]}
      />
    </svg>
  );
};

export default Example;
```

- Every `<path>` uses `fill={COLORS[color]}`. No `stroke`, and no hardcoded colors.
- Use `fillRule="evenodd"` and `clipRule="evenodd"` on paths with cut-outs.
- Default `color` is `ui-fleet-black-75` unless the icon has a fixed meaning (`error` defaults to `status-error`, `success` to `status-success`).
- Keep the `viewBox` at `0 0 16 16` and size with `ICON_SIZES`. Don't change the drawing per size.
- Register it in `ICON_MAP` in `frontend/components/icons/index.ts` (import and map entry, kebab-case key).

## 5. Use it

- Render with `<Icon name="example" />`. Pass `color` only when it differs from the default.
- In buttons, use `<Button icon="example">`, never an `<Icon>` child. Icon-only buttons need `ariaLabel`.
- An icon next to text in a tooltip should sit centered on the line. `TooltipWrapper` already centers a lone icon.

## Software and OS logo tiles

Fallback software icons and OS logos live in `frontend/pages/SoftwarePage/components/icons/` and render as tiles (`viewBox="0 0 32 32"`):

- Background `#F9FAFC` (`<path fill="#F9FAFC" d="M0 0h32v32H0z" />`), logo in `#515774`.
- On 32px tiles the logo sits in an 18.67px area, the same size for every logo.
- 24px tiles (lists) use the 16px drawing at its native size. See `MacOS.tsx`, which checks `props.width === "24"`.
- Brand logos for Fleet-maintained apps use the app's own logo (PNG) and aren't redrawn.

## 6. Verify

- Run `npx prettier --check` and `npx eslint` on the changed files.
- Run related tests with `yarn test --maxWorkers=2 --findRelatedTests <files>`. Don't run the full suite.
- Check the icon in the running app at 100% zoom, in light context, next to the text it sits beside.
- For user-visible changes, add a changes file in `changes/` and attach before/after screenshots to the PR.

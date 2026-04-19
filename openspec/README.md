# OpenSpec example for Fleet story #38785

Strict OpenSpec format. Illustrative example of what a full spec would look like if written before
implementation of story fleetdm/fleet#38785.

## Layout

```
openspec/
└── changes/
    └── windows-setup-cancel-if-software-fails/
        ├── proposal.md     # Intent, Scope, Approach
        ├── design.md       # Technical approach bullets
        ├── tasks.md        # Numbered categories with N.M task items
        └── specs/
            └── mdm-windows-setup-experience/
                └── spec.md # ADDED / MODIFIED / REMOVED requirements with GIVEN/WHEN/THEN scenarios
```

## How strict OpenSpec answers the six behavioral contract questions

| Question                          | Answered in                                                          |
|-----------------------------------|----------------------------------------------------------------------|
| What goes in?                     | spec.md, GIVEN clauses                                               |
| What comes out?                   | spec.md, THEN clauses                                                |
| What must stay true?              | spec.md, SHALL statements plus MODIFIED Requirements                 |
| What side effects?                | spec.md, scenarios for activity emission and DB state changes        |
| How does it fail?                 | spec.md, timeout and failed install scenarios                        |
| What is explicitly out of bounds? | Not expressible. Strict OpenSpec has no non-goals section.           |
| Which packages / files?           | Not expressible. Strict design.md is decisions only, not file lists. |

The last two gaps are the honest cost of staying strict. Teams that want non-goals and package lists have to
extend the format (or pair it with an ADR).

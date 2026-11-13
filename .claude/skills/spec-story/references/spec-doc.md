# Spec doc

The spec doc is the reference for the engineers who build the story. It holds what the GitHub bodies leave out: checked code anchors, code shapes, migration plans, and the reason behind each decision. Write it for an engineer who joins halfway through and needs to know why the code will look the way it does.

Follow the style rules in SKILL.md 3.1.


## Sections

Include every section unless it's marked "if needed." Write "None." in an empty section instead of dropping it.

1. **Story.** Quote the user story (`As a... I want... so that...`). Then explain it in 2 to 4 short paragraphs. Lead with the scope calls as bold callouts, each with its reason, like `**ADE only.** <why>` or `**Fleet Premium only.** <why>`.
2. **Feature design.** Include the Mermaid sequence diagram from Stage 1, updated. Give every real actor a lane, like `Admin`, `UI`, `API`, `EE`, `DB`, `Worker`, or `Device`. Put real function and endpoint names on the arrows. Mark phases with `note over X,Y: <n>. <phase>`.
3. **Figma copy and dev notes** (if needed). Add one `Element | Text` table per surface, like a page, modal, dropdown, or activity feed. Copy every string word for word, punctuation included. That means titles, labels, tooltips, descriptions, flash messages, errors, and disabled-state tooltips. List the dev notes under each table, such as who sees what, platform limits, GitOps mode, defaults, and caps.
4. **Expert review notes.** List SME feedback as numbered notes, with the SME's name on the heading line. Keep the notes even after you've used them. If you skip SME review, give the reason in one line.
5. **Design review and Verification.** Include the Stage 2 results, updated by the Stage 3 check.
6. **Engineering answers.** Copy every item under the parent story's `### Engineering` and `### Risk assessment` headings as written, and answer each one. Work from the parent, not from memory, since the story template changes. These answers go back into the parent in Stage 4.
7. **API design note** (if needed). When more than one doc PR shapes the API or YAML, add a `PR | Endpoint or key | What changes` table. List each conflict and how it's settled. Note any field that's named differently in the REST API and GitOps YAML on purpose, and how the `renameto` struct tag maps them.
8. **Deep dives** (if needed). Give each tricky topic its own H2, named for the subject, like "Password delivery to the device." Include diagrams, payload examples, and struct shapes. Explain why this design beat the other options.
9. **Sub-issue summary.** Add a `# | Title | Layer | Depends on` table.
10. **Sub-issues.** Write each one in the format below. The docs and QA sub-issue goes last.
11. **Dependency graph.** Draw it in ASCII, with `┌──►` and `└──►` for parallel branches. Under it, explain the critical path in 1 to 3 sentences, and say what can start from the API contract.
12. **PR plan.** Add a `PR | Sub-issues | Can start after | Parallel with` table.
13. **Team plans.** Draw Gantt-style ASCII for 2 engineers, with one backend track and one frontend track. Add a 3-engineer plan if the story can use one. You need this for 4 or more sub-issues.
14. **Shared sub-issues** (if needed). When two people could share one sub-issue on one branch, add a `Piece | Files | Can start` table.
15. **Follow-ups.** List related work you kept out of scope, like nearby bugs, refactors, and nice-to-haves. Write "None." if there aren't any.
16. **Open questions.** For each one, say who decides and what they need to decide.
17. **Resolved questions.** List each decision made during review, with its reason. Add a note on future changes when it helps.
18. **Testing tips** (if needed). Write numbered CLI steps for checking the feature on a real device, like `fleetctl api ...`, `dscl ...`, or `gh issue view ...`.


## Sub-issue format

```markdown
## Sub-issue N: <feature>: <area>

**Related user story:** #<parent>
**Depends on:** <numbers, or "none">
**Parallel with:** <numbers, or "none">

<2 to 4 sentences on what this sub-issue ships.>

### Task

<How to build it. See below.>

### Condition of satisfaction

<A checklist you can test. See below.>
```

Start every title with the same short feature name, then the area, like `Managed local account: DB migration, types, datastore, and MDM command primitives`. The shared name groups the issues in the tracker.


### Task

The Task tells an engineer how to build it. Don't restate the story. Include:

- **A `####` heading per work area,** like "Enrollment worker," "MDM ack handler," or "API endpoint," for anything longer than a page.
- **Checked anchors,** like `server/worker/apple_mdm.go:237`, plus the function or symbol name, since line numbers drift. Confirm each one with Read or Grep. Never guess.
- **Code blocks** for every change that isn't trivial, like Go structs and signatures, SQL, TypeScript, plists or XML, and JSON requests and responses.
- **Patterns to copy,** like "follow `getHostRecoveryLockPasswordEndpoint` in `server/service/hosts.go`." One pointer to a pattern saves pages of explanation.
- **The reason behind each choice,** including the option you didn't take. Use bold headings like "**Why a new table, not a column on `host_recovery_key_passwords`:**" so nobody reopens settled decisions.
- **Edge cases and guards,** like auth and permission checks, platform checks, nil checks, license checks, and hosts that enrolled before the feature. Also flag any ack handler that has to tell this command apart from another use of the same MDM command.
- **Follow-up commands,** like `make generate-mock` after a datastore interface change.
- **A migration plan** when the sub-issue adds a migration. See SKILL.md 3.1.

If a Task reads like a to-do list ("add the endpoint," "add the migration"), it needs more detail.


### Condition of satisfaction

Write a checklist grouped by surface or scenario, under bold labels like "**Enrollment and ack:**", "**API:**", or "**End to end:**". Include:

- Test commands with env vars, like `MYSQL_TEST=1 go test ./server/datastore/mysql/...`, `MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/...`, or `yarn test`.
- Checks with real inputs and outputs.
- Negative cases, like the wrong platform, Fleet Free, hosts that enrolled before the feature, and GitOps mode.
- An end-to-end test when the flow spans systems.
- Exact-match checks where output has to stay the same, like an MDM payload that mustn't change when the feature's off.


## Quality bar

#47904 (App Store app versions) is the reference story. Its sub-issues, #53637 through #53642, show the GitHub side. Its spec doc isn't public, but it followed the sections above. A draft is ready when:

- An engineer could start any sub-issue from the spec doc alone, without redoing the research.
- Every claim about today's code has a checked anchor.
- Every choice that differs from the obvious option says why.
- Every doc PR conflict and SME point is settled or listed as an open question.

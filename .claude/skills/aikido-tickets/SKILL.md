---
name: aikido-tickets
description: Create GitHub issues in fleetdm/confidential for Aikido pen test findings. Use when asked to "create aikido tickets", "aikido ticket", or "create pen test tickets".
allowed-tools: Bash(gh *), Read, Grep, Glob, WebFetch
model: sonnet
effort: high
---

# Create Aikido Pen Test Tickets

Create GitHub issues in `fleetdm/confidential` for Aikido penetration test findings.

## Prerequisites

Before starting, walk the user through these steps:

### 1. Export the Aikido report

The user needs to download the pen test report PDF from Aikido:

1. Go to the Aikido assessment page (e.g., `app.aikido.dev/ai-pentests/projects/.../assessments/.../issues`)
2. Click the purple **"Download Report"** button in the top-right corner
3. Select **"Detailed Auditor Report"** (this contains every finding with technical details and remediation steps)
4. Click **Continue** and save the downloaded PDF
5. Provide the path to the downloaded PDF

### 2. GitHub project board permissions

Adding issues to project boards requires the `project` scope on the GitHub CLI token. Test by running:

```bash
gh project list --owner fleetdm --limit 1
```

If this fails with a scope error, the user needs to run interactively:

```bash
gh auth refresh -s project -h github.com
```

**Alternative:** If the user cannot or prefers not to grant project scope, skip the project board step. The issues will still be created with correct labels and assignment. The user (or their manager) can manually drag them into the correct project board column afterward.

## Inputs

Ask the user for these if not provided:

- **Pen test PDF report path:** Path to the downloaded Aikido detailed auditor report PDF
- **PT ID(s):** Which PT-* findings to create tickets for (specific IDs, a range, or "all")
- **Team:** Which team owns the findings (determines labels, project board, and parent story)
- **Assignee:** GitHub username to assign the tickets to
- **Parent story:** The confidential issue number that tracks these findings (e.g., #16715)

## Teams and project boards

| Team | Label | GitHub Project |
|------|-------|----------------|
| Orchestration | `#g-orchestration` | https://github.com/orgs/fleetdm/projects/71/ |
| Security & Compliance | `#g-security-compliance` | https://github.com/orgs/fleetdm/projects/97/ |
| MDM | `#g-mdm` | https://github.com/orgs/fleetdm/projects/58/ |
| Software | `#g-software` | https://github.com/orgs/fleetdm/projects/70/ |
| First Impressions | `#g-first-impressions` | https://github.com/orgs/fleetdm/projects/105/ |

If the user specifies a team not listed here, ask for the team label and project URL/number.

## Ticket format

### Title
```
Aikido-PT-{number} [{SEVERITY}]: {concise title from the finding}
```

### Body structure

```markdown
## {concise title}

{1-2 sentence explanation of what is wrong}

See full details below.

### Attack path

{Concise but complete description of how an attacker exploits this vulnerability}

### Fix

**Option 1 (recommended):** {Best fix approach described concisely}

{Only add Option 2, Option 3 etc. if there are genuinely different viable approaches. If one fix is clearly best, only list that one.}

---

**Aikido ref:** PT-{number} | CVSS {score} | `{primary affected file}`
**Parent story:** #{parent_story_number}

---

<details>
<summary>Aikido pen test details (PT-{number})</summary>

{Full content from the Aikido PDF report for this finding, including:}
### Description
{original description}

### Business impact
{original business impact}

### How to exploit
{all exploit steps with code blocks}

### Remediation
{all remediation bullets from the report}

### References
{CVSS score and vector}

</details>
```

## Process

1. **Read the finding** from the pen test PDF report. Each finding starts with a header like "2.3.X PT-{N} - {Title}". The detailed findings section starts around page 15. Estimate ~3 pages per finding to locate the right page range.

2. **Write the ticket body** following the format above:
   - The top section (title, explanation, attack path, fix) is YOUR synthesis of the finding, written concisely
   - The foldable `<details>` section at the bottom contains the ORIGINAL Aikido report content verbatim

3. **Create the issue:**
   ```bash
   gh issue create --repo fleetdm/confidential \
     --title "Aikido-PT-{N} [{SEVERITY}]: {title}" \
     --assignee {assignee} \
     --label "bug,~security,~vulnerability-management,{team_label},p3" \
     --body "{body}"
   ```

4. **Add to the correct project board and set status to Ready:**
   ```bash
   # Get project node ID (needed for item-edit)
   PROJECT_NODE_ID=$(gh api graphql -f query='{ organization(login: "fleetdm") { projectV2(number: {N}) { id } } }' | jq -r '.data.organization.projectV2.id')

   # Add to project
   ITEM_ID=$(gh project item-add {project_number} --owner fleetdm --url {issue_url} --format json | jq -r '.id')

   # Get Status field ID and Ready option ID
   gh project field-list {project_number} --owner fleetdm --format json | jq '.fields[] | select(.name == "Status")'

   # Set to Ready (option ID for "Ready" is typically 567b541e but verify from field-list output)
   gh project item-edit --project-id $PROJECT_NODE_ID --id $ITEM_ID --field-id {status_field_id} --single-select-option-id {ready_option_id}
   ```

   If the project scope is not available, inform the user that tickets were created but need to be manually added to the project board.

5. **Report** the created issue URL to the user.

## Batch creation

When creating many tickets at once, use parallel agents (5-6 at a time, each handling ~5-7 findings) for efficiency. Each agent should:
- Read the relevant PDF pages for its assigned findings
- Create the issues with correct labels and assignee
- Add each issue to the project board and set status to Ready

## Important notes

- All tickets go in `fleetdm/confidential` (private repo) since they contain security findings
- Always use `p3` priority label unless the user specifies otherwise
- Always use `bug` label (not `story`)
- The foldable `<details>` section preserves the full Aikido evidence for reference
- When creating many tickets, read the PDF pages for each finding to get accurate details
- For the Fix section: if one fix is clearly the best, only list that one. Only list multiple options if there are genuinely different viable approaches
- Fetch project field IDs at runtime rather than hardcoding, since they may change

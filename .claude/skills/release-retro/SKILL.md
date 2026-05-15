---
name: release-retro
description: Format release retro notes for a Fleet working group into a Slack recap post and ~timebox GitHub issues. Use when asked to "post retro recap", "format retro notes", "create release retro", or after a retrospective when feedback needs to land in Slack and action items need to be tracked.
allowed-tools: Bash(gh *), Bash(grep *), Bash(date *), Read, Grep, Glob, mcp__claude_ai_Slack__slack_search_channels, mcp__claude_ai_Slack__slack_search_users, mcp__claude_ai_Slack__slack_list_channel_members, mcp__claude_ai_Slack__slack_send_message_draft
model: sonnet
effort: medium
---

# Release retro: format and post

Process a working group's release retro notes into (1) one Slack draft summarizing the feedback, and (2) one `~timebox` GitHub issue per action item.

## Inputs

Invoked without arguments. Prompt the user for each input in sequence:

1. **Working group's Slack channel** (e.g. `g-first-impressions`, `g-power-to-pc`). Accept with or without a leading `#`; normalize internally.
2. **Release version** covered (e.g. `4.86.0`).
3. **Retro notes**: ask the user to paste the raw notes. If the notes are long, accept them across multiple messages and confirm when they're done.

Retro notes typically have two parts. Both may appear in any format the user pastes:

- **Feedback** — often broken out per attendee (Wins / What went well, Friction / What could have gone better, Things to remember). Your job is to *synthesize across people into thematic bullets*. Do not list per person in the recap.
- **Action items** — usually prefixed `TODO <Name>:` or similar. Each becomes one GitHub issue with the named people as assignees.

## Process

### 1. Resolve the working group

- Slack channel ID: call `slack_search_channels` with the channel name; pick the matching result.
- User group ID: the working group has a Slack user group whose handle usually mirrors the channel (e.g. `first-impressions-group` for `#g-first-impressions`, `power-to-pc-group` for `#g-power-to-pc`). Find it via `slack_list_channel_members` with `include_bots=true` or `slack_search_users`. User group IDs start with `S`. If you cannot resolve it, ask the user.
- GitHub label: matches the channel name with a `#` prefix (e.g. `#g-first-impressions`). Verify with `gh label list --repo fleetdm/fleet --search "<channel-name>"`.

### 2. Resolve GitHub handles

For each name mentioned in an action item, find their GitHub handle:

```bash
grep -r "<Full Name>" handbook/ | grep -i "github"
```

The handle appears as `_([@handle](https://github.com/handle))_`. If you cannot find it, ask the user.

### 3. Resolve relative dates

If the notes say "by Wednesday" or "next Monday," anchor against today via `date -u +%Y-%m-%d` and convert to absolute dates in the issue body so the timebox stays interpretable later.

### 4. Draft the GitHub issues

For each action item, draft an issue using the timebox template (`.github/ISSUE_TEMPLATE/timebox.md`):

```
## Related user story
<one-sentence user story framing>

## Task
<the action item rewritten as a clear task; preserve collaborators and deadlines>

## Condition of satisfaction
<concrete completion criteria, with absolute dates>
```

- Labels: `~timebox` + the working group's channel label (e.g. `#g-first-impressions`).
- Assignees: every named person on the action item.

### 5. Draft the Slack recap

Template:

```
:recycle:  <version> release retro recap <@<USER_GROUP_ID>|<group-handle>>

<one short framing sentence>. Quick summary:

Wins
• <bulleted, synthesized across attendees>

Friction
• <bulleted, synthesized across attendees>

Themes worth calling out
• <narrative or named themes pulled from "Things to remember" / "Themes" sections>

Action items
• <github issue link 1>
• <github issue link 2>
• ...

Thanks team for the honest feedback.
```

Style:
- Casual, conversational, first-person plural ("we").
- Short, punchy bullets.
- **No em dashes.** Use periods, commas, colons, or parentheses instead, even if the source notes use em dashes.
- Slack link syntax: `<https://...|text>`.
- Issue links: `<https://github.com/fleetdm/fleet/issues/N|#N: short title>`.

Section headings may be renamed to fit the cycle (e.g. "What went well" instead of "Wins"). Keep the order.

### 6. Show the user for review BEFORE creating anything

Present:
1. The list of issues you plan to create: title, assignees, labels, full body.
2. The full Slack draft text.

When previewing the Slack draft in chat, render user-group and user mentions as friendly handles (e.g. `@first-impressions-group`, `@allenhouchins`), NOT the raw `<@ID|handle>` form. Use the canonical raw form in the actual API call.

Wait for explicit confirmation.

### 7. Create the issues and Slack draft

On confirmation:

1. Create the GitHub issues in parallel:
   ```bash
   gh issue create --repo fleetdm/fleet \
     --title "<title>" \
     --label "~timebox,<channel-label>" \
     --assignee <handle> [--assignee <handle> ...] \
     --body "<timebox-template-body>"
   ```
2. Replace the action-item placeholders in the Slack draft with the real issue numbers and titles.
3. Save the Slack post as a draft (not sent) via `slack_send_message_draft` with `channel_id=<resolved channel ID>`.
4. Report back: list of issue URLs, channel link for the draft.

## Constraints

- **Default to drafting, never sending.** Only post to Slack if the user explicitly says to.
- **Never use `@channel` or `@here`.** Only the proper Slack user group.
- **No em dashes** in the recap, even if source notes use them.
- If an action item lacks an obvious assignee, ask before creating.
- If a name doesn't resolve to a GitHub handle from the handbook, ask before creating.

## Example

For the 4.86.0 retro in `#g-first-impressions`, the resolved IDs were:
- Channel: `C0ACJ8L1FD0`
- User group: `S0AR86XUSSZ` (`first-impressions-group`)
- Label: `#g-first-impressions`

For `#g-power-to-pc`:
- Channel: `C0AQY8D7FM4`
- User group: `S0ARQGHL457` (`power-to-pc-group`)
- Label: `#g-power-to-pc`

These are cached pointers; always re-verify with the Slack API in case they change.

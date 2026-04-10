---
name: project
description: Load or initialize a Fleet workstream project context. Use when asked to "load project" or "switch project".
context: fork
allowed-tools: Read, Write, Glob, Grep, Bash(ls *), Bash(pwd *)
effort: medium
---

# Load a workstream project context

## Detect the project directory

Find the Claude Code auto-memory directory for this project. It's based on the working directory path:

1. Run `pwd` to get the current directory.
2. Construct the memory path: `~/.claude/projects/` + the cwd with `/` replaced by `-` and leading `-` (e.g., `/Users/alice/Source/github.com/fleetdm/fleet` → `~/.claude/projects/-Users-alice-Source-github-com-fleetdm-fleet/memory/`).
3. Verify the directory exists. If not, tell the user and stop.

Use this as the base for all reads and writes below.

## Load the project

Look for a workstream context file named `$ARGUMENTS.md` in the memory directory. This contains background, decisions, and conventions for a specific workstream within Fleet.

If the project context file was found, give a brief summary of what you know and ask what we're working on today.

If the project context file doesn't exist:
1. Tell the user no project named "$ARGUMENTS" was found.
2. List any existing `.md` files in the memory directory so they can see what's available.
3. Ask if they'd like to initialize a new project with that name.
4. If they don't want to initialize, stop here.
5. If they do, ask them to brain-dump everything they know about the workstream — the goal, what areas of the codebase it touches, key decisions, gotchas, anything they've been repeating at the start of each session. A sentence is fine, a paragraph is better. Also offer: "I can also scan your recent session transcripts for relevant context — would you like me to look back through recent chats?"
6. If they want you to scan prior sessions, look at the JSONL transcript files in the Claude project directory (the parent of the memory directory). Read recent ones (last 5-10), skimming for messages related to the workstream. These are large files, so read selectively — check the first few hundred lines of each to gauge relevance before reading more deeply.
7. Using their description, any prior session context, and codebase exploration, find relevant files, patterns, types, and existing implementations related to the workstream.
8. Create the project file in the memory directory using this structure:

```markdown
# Project: $ARGUMENTS

## Background
<!-- What is this workstream about, in the user's words + what you learned -->

## How it works
<!-- Key mechanisms, patterns, and code flow you discovered -->

## Key files
<!-- Important file paths for this workstream, with brief descriptions -->

## Key decisions
<!-- Important architectural or design decisions -->

## Status
<!-- What's done, what remains -->
```

9. Show the user what you wrote and ask if they'd like to adjust anything before continuing.

As you work on a project, update the project file with useful discoveries — gotchas, important file paths, patterns — but not session-specific details.

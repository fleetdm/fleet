Read the project context file at `~/.fleet/claude-projects/$ARGUMENTS.md`. This contains background, decisions, and conventions for a specific workstream within Fleet.

Also check for a project-specific memory file named `$ARGUMENTS.md` in your auto memory directory (the persistent memory directory mentioned in your system instructions). If it exists, read it too — it contains things learned while working on this project in previous sessions.

If the project context file was found, give a brief summary of what you know and ask what we're working on today.

If the project context file doesn't exist:
1. Tell the user no project named "$ARGUMENTS" was found.
2. List any existing `.md` files in `~/.fleet/claude-projects/` so they can see what's available.
3. Ask if they'd like to initialize a new project with that name.
4. If they don't want to initialize, stop here.
5. If they do, ask them to brain-dump everything they know about the workstream — the goal, what areas of the codebase it touches, key decisions, gotchas, anything they've been repeating at the start of each session. A sentence is fine, a paragraph is better. Also offer: "I can also scan your recent session transcripts for relevant context — would you like me to look back through recent chats?"
6. If they want you to scan prior sessions, look at the JSONL transcript files in the Claude project directory (the same directory as your auto memory, but the `.jsonl` files). Read recent ones (last 5-10), skimming for messages related to the workstream. These are large files, so read selectively — check the first few hundred lines of each to gauge relevance before reading more deeply.
7. Using their description, any prior session context, and codebase exploration, find relevant files, patterns, types, and existing implementations related to the workstream.
8. Create `~/.fleet/claude-projects/$ARGUMENTS.md` populated with what you found, using this structure:

```markdown
# Project: $ARGUMENTS

## Background
<!-- What is this workstream about, in the user's words + what you learned -->

## How It Works
<!-- Key mechanisms, patterns, and code flow you discovered -->

## Key Files
<!-- Important file paths for this workstream, with brief descriptions -->

## Key Decisions
<!-- Important architectural or design decisions -->

## Status
<!-- What's done, what remains -->
```

9. Show the user what you wrote and ask if they'd like to adjust anything before continuing.

As you work on a project, update the memory file (in your auto memory directory, named `$ARGUMENTS.md`) with useful discoveries — gotchas, important file paths, patterns — but not session-specific details.

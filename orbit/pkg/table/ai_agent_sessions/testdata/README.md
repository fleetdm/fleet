# Test data

`cursor/` is a real Cursor CLI (`cursor-agent` 2026.10.22-27dcf5f) capture from a Linux VM, taken 2026-10-27, laid out as under `~/.cursor`. The user name and home directory were replaced with `alice`, and the `chats/` directory renamed to match (it is the MD5 of the session's working directory). Sign-in files were never captured.

- `d038b33d-…`: one session of nine prompts that read a file, list files, search, run commands (`cat .env.local | grep API_`, `ls -la`, `false`), write a file, fetch a URL (Cursor's `WebFetch` through `CallDynamicTool`) and call an MCP server's tool (`everything`'s `echo`).
- `422caa19-…`: a short session the user interrupted while a command was running.

The other agents' tests use small hand-written fixtures, built from each agent's source code or published schema.

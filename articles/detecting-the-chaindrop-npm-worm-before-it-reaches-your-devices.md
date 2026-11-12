# Detecting the ChainDrop npm worm before it reaches your devices

*ChainDrop spreads by hiding in the npm packages and IDE config files your developers already trust. Here is how to see whether it has landed on your fleet before a stolen credential tells you it has.*

## Key takeaways

- **You can check your fleet today instead of waiting for the breach notification.** Fleet's agent already queries installed npm packages and IDE configuration files across every host, so you can ask "is a compromised package or a hidden auto-run hook on any of our machines?" right now, not after a leaked credential surfaces the answer for you.
- **ChainDrop self-propagates through developer tooling, which is exactly why signature checks miss it.** It has already compromised more than 1,300 packages and spreads further by planting hooks in Claude Code and VS Code that fire automatically when a teammate opens a poisoned repository. Nothing about that looks like an attack to an endpoint protection agent.
- **The `npm_packages` table only sees global installs, so a live query alone leaves a gap.** It never walks the project-local `node_modules/` directories where compromised dependencies actually live. Pair the query with a filesystem deep scan through Fleet's run-script to close it.
- **The malicious hooks live in files you can already read across the fleet.** VS Code folder-open tasks and Claude Code settings are plain config files. Fleet reads them across every user on every host, so the auto-run persistence ChainDrop relies on becomes queryable.
- **Detection becomes a standing policy, not a one-time sweep.** Save the hunt as a Fleet policy and every new host, and every host that changes, gets checked automatically against the current known-bad list.
- **Rotate first if you find it, investigate second.** A worm built to steal developer credentials has likely already used them. Treat any hit as an exposure and start the rotation clock in minutes.

<a purpose="cta-button" href="/contact">Talk to Fleet security</a>

ChainDrop is a new self-propagating npm supply chain worm, and it has already compromised more than 1,300 packages. What makes it worth your attention is not the raw count, it is how it moves. Beyond the usual trick of riding along on `npm install`, ChainDrop plants malicious hooks in developer tooling, specifically in Claude Code and VS Code, that run automatically the moment a teammate opens a poisoned repository. One developer pulls a compromised branch, opens it in their editor, and the worm executes without anyone typing a command.

That is the part that should worry a security team. A worm that spreads through the tools your engineers use all day, using mechanisms those tools ship on purpose, does not trip the alarms built to catch malware. The signatures are valid, the config files are legitimate file formats, and the editor is doing exactly what it was told to do. You do not detect this by waiting for a malicious-behavior alert. You detect it by inventorying what is actually on the machine. Fleet's agent already collects that inventory, so the question becomes: has a compromised package or a hidden auto-run hook landed on your fleet? Here is how to answer it.

## What ChainDrop does and why it spreads through IDEs

Classic npm supply chain attacks depend on a developer running `npm install` against a poisoned package. ChainDrop does that too, but it adds a second, quieter propagation path: it drops hooks into developer tooling that execute on their own, without an install step.

The mechanism it abuses is a feature, not a bug. Modern editors and AI coding tools support hooks and tasks that run automatically when you open a folder or start a session. That is genuinely useful when you wrote the config yourself. It is a loaded gun when the config arrives inside a repository you just cloned. When a teammate opens a poisoned repository, the planted hook runs with that user's privileges, in that user's environment, next to that user's credentials, and the worm keeps moving.

This is why standardizing on one editor or one AI vendor does not contain the risk. The auto-run surface exists across the whole developer toolchain, and the worm only needs one poisoned repo to reach one trusting teammate. Detection has to look at the tooling itself, not just at the npm dependency tree.

## Where ChainDrop can hide on a host

Before writing queries, it helps to name the three places a host can be carrying ChainDrop, because each one needs a different detection approach.

**Global npm packages.** Packages installed globally live in a handful of well-known paths, and Fleet's agent can read them directly through the `npm_packages` table. This is the fastest place to check and the easiest to miss things in, for reasons below.

**Project-local `node_modules/`.** This is where the vast majority of npm packages actually live: inside each project's own `node_modules/` directory. The `npm_packages` table does not walk these, so covering them takes a filesystem scan.

**IDE and AI-tool auto-run hooks.** The persistence and lateral-movement half of ChainDrop lives in config files: VS Code folder-open tasks and settings, and Claude Code session hooks. These are the files that turn "someone cloned a bad repo" into "code executed on their machine."

You need to check all three. A host can return a clean npm inventory and still be carrying a planted editor hook, and vice versa.

## Finding compromised npm packages with Fleet

Start with the fast, fleet-wide check. Fleet's `npm_packages` table reports globally installed packages, and a live query surfaces any match against the known-bad list in seconds across every host:

```sql
SELECT name, version, path FROM npm_packages
WHERE name IN (
  -- replace with the current ChainDrop compromised-package list
);
```

The `WHERE name IN (...)` list is the one thing you have to supply, and it is the one thing you should not hard-code and forget. A self-propagating worm keeps compromising new packages, so any static list goes stale fast. Pull the current set from Fleet's maintained compromised-npm-packages policy and list, which is continuously updated, and treat that as the authoritative source rather than a snapshot you pasted in last week. Version matters too: for many campaigns only specific versions of a package are malicious, so match on name and version together when your source gives you that precision.

### Mind the global-only gap

Here is the blind spot every team relying on `npm_packages` should know about. As we found documenting a previous npm worm, the `npm_packages` table only covers global installs. Fleet's agent looks at default global paths like `~/.npm-global`, `/usr/local/lib/node_modules`, and `/opt/homebrew/lib/node_modules`. It does not walk into project-local `node_modules/` directories.

In practice, almost no one installs their dependencies globally. A developer with twenty active projects, each carrying a compromised package inside its own `node_modules/`, will return zero rows from the query above. The live query is a real and useful global exposure check, but on its own it will tell you a badly infected machine is clean.

### Close the gap with a deep-scan script

To cover project-local installs, pair the live query with a filesystem deep scan run through Fleet's run-script. The approach is the same one we shipped for the earlier worm: enumerate every `node_modules/` directory under user home directories, then inspect each package's manifest.

```bash
#!/bin/sh
# Deep scan for ChainDrop-compromised packages in project-local node_modules.
# Populate BAD_PACKAGES from Fleet's maintained compromised-npm-packages list.
BAD_PACKAGES="/tmp/chaindrop_bad_packages.txt"   # one "name@version" per line

find "$HOME" -maxdepth 10 -type d -name node_modules 2>/dev/null | while read -r nm; do
  find "$nm" -maxdepth 2 -name package.json 2>/dev/null | while read -r pkg; do
    name=$(sed -n 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$pkg" | head -1)
    version=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$pkg" | head -1)
    if grep -qxF "${name}@${version}" "$BAD_PACKAGES" 2>/dev/null; then
      echo "COMPROMISED: ${name}@${version} at ${pkg}"
    fi
  done
done
```

Run it through Fleet's run-script feature and you get per-host filesystem coverage that reaches the `node_modules/` directories the live query cannot. The two layers are complementary: the live query gives you a fast global answer in seconds, and the script gives you complete visibility. Run both. As with the package list, the deep scan's known-bad file should be filled from Fleet's maintained list, not from a hard-coded set, so it stays current as the worm spreads.

## Finding the malicious IDE and Claude Code hooks with Fleet

The npm checks find compromised code. They do not find the auto-run hooks that let ChainDrop execute the moment a teammate opens a poisoned repo. Those live in editor and AI-tool config files, and Fleet reads those files across every user on every host.

First, inventory what editors and extensions are even in play. The `vscode_extensions` table is uid-scoped, which means it returns data for the current user by default and needs a `CROSS JOIN` against the `users` table to cover everyone on the host. The `CROSS JOIN` matters specifically: it tells the query planner not to reorder the tables, which is what produces the empty-result gotcha on uid-scoped tables when the query runs as root.

```sql
SELECT u.username, e.name, e.publisher, e.version, e.vscode_edition
FROM users u CROSS JOIN vscode_extensions e USING (uid);
```

Next, read the config files where the auto-run hooks would live. The kind of hook ChainDrop plants shows up in a VS Code folder-open task, a `.vscode/tasks.json` entry with `"runOn": "folderOpen"`, or in editor and Claude Code settings like `.vscode/settings.json`, `.claude/settings.json`, and `.claude.json`. Fleet can pull the contents of those files across every user by joining the `users` table to `file_lines`, then parse the JSON with `json_each`:

```sql
WITH path_suffixes(path) AS (
  VALUES
    ('/.vscode/tasks.json'),       -- VS Code tasks (folderOpen auto-run)
    ('/.vscode/settings.json'),    -- VS Code settings
    ('/.claude/settings.json'),    -- Claude Code settings
    ('/.claude.json')              -- Claude Code config
),
full_paths AS (
  SELECT u.directory || p.path AS full_path, p.path AS suffix
  FROM users u
  JOIN path_suffixes p ON 1=1
),
config_files AS (
  SELECT f.path, group_concat(f.line, '') AS contents
  FROM file_lines f
  JOIN full_paths fp ON f.path = fp.full_path
  GROUP BY f.path
)
SELECT cf.path, je.key AS setting, je.value AS value
FROM config_files cf
JOIN json_each(cf.contents) AS je;
```

From there you are reading real settings out of real files. A VS Code task whose `runOn` is `folderOpen` and whose command shells out to `curl`, `node -e`, or a script inside the repo is the pattern to flag. The same goes for a Claude Code settings entry that wires up a hook to run on session start. To be precise about what this catches: these are the auto-run mechanisms an attacker abuses, the kind of hook ChainDrop plants, not a published, verified ChainDrop indicator. You are hunting for the behavior, which is what makes it durable when the specific package names churn.

If you would rather hunt by presence than by contents, the `file` table finds the config and hook files by path across hosts, so you can see which machines even have a `.vscode/tasks.json` or a `.claude/settings.json` before you dig into what is inside them:

```sql
SELECT path FROM file
WHERE path LIKE '/Users/%/.vscode/tasks.json'
   OR path LIKE '/Users/%/.claude/settings.json'
   OR path LIKE '/Users/%/.claude.json';
```

Swap in the home-directory prefixes that match your fleet's operating systems.

## Turning detection into continuous policies

Running these checks once tells you about today. A self-propagating worm is a moving target, so the more useful posture is a standing one. Any of the queries above can become a Fleet policy: the package-match live query, the config-file hook query, the file-presence check. Saved as policies, they run on a schedule and against every host that enrolls or changes, so a machine that pulls a poisoned repo next week gets flagged without anyone re-running the hunt.

Because Fleet is GitOps-native, those policies live in a Git repository as YAML, get reviewed in a pull request, and deploy through CI. When Fleet's maintained compromised-package list adds new entries, you update the policy the same way you update any code: a reviewable, reversible change, not an undocumented console edit. And everything the queries surface rolls into Fleet's software inventory, so the same package data feeds vulnerability matching and reporting across the fleet.

## What to do if you find it

Treat a hit as an exposure, not a curiosity. ChainDrop is built to steal developer credentials and keep spreading, so a compromised package or a planted hook on a machine means you should assume the credentials on that machine are already at risk.

Move in this order. Isolate any host with a confirmed compromised package or a suspicious auto-run hook. Rotate the credentials that a developer machine tends to hold, npm tokens, source-control access tokens, cloud provider keys, and any secrets reachable from the shell that hook would have inherited. Remove the offending package or hook, and because the worm plants persistence in editor config, check that the hook has not been re-added. Then widen the search: audit the repositories that machine touched, since the whole point of the IDE-hook path is that opening a poisoned repo is enough to spread. Finally, keep the policies from the previous section running so re-infection surfaces immediately instead of on the next manual sweep.

Rotation comes before forensics. If a worm designed to exfiltrate credentials reached a machine, the safe assumption is that it already did its job, and the rotation clock should be measured in minutes.

## The window is open now

ChainDrop's advantage is speed and silence: it spreads through tooling your developers trust, using features those tools ship on purpose, and it does its damage before anyone files a ticket. Your advantage is that the same tooling leaves a footprint on disk, and that footprint is queryable right now. You do not have to wait for a stolen credential to appear on a paste site to learn whether a compromised package or a hidden hook is on your fleet. You can ask, across every host, in seconds, and then keep asking automatically.

*See whether ChainDrop has reached your devices. [Talk to Fleet security](https://fleetdm.com/contact), and explore Fleet's [reports library](https://fleetdm.com/reports) and [GitOps documentation](https://fleetdm.com/docs/configuration/yaml-files) to turn detection into standing policy.*

---

About the author: [Dhruv Majumdar](https://www.linkedin.com/in/neondhruv) is Fleet's VP of Security Solutions. Talk to [Fleet](https://fleetdm.com/device-management) today to find out how to solve your trickiest device management, data orchestration, and security problems.

<meta name="articleTitle" value="Detecting the ChainDrop npm worm before it reaches your devices">
<meta name="authorFullName" value="Dhruv Majumdar">
<meta name="authorGitHubUsername" value="karmine05">
<meta name="category" value="security">
<meta name="publishedOn" value="2026-08-17">
<meta name="description" value="ChainDrop hides in npm packages and IDE hooks your developers trust. Use Fleet to see if it is on your devices before a stolen credential does.">

</content>

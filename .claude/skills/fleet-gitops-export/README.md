# Export your Fleet to GitOps

This skill lets an AI coding agent turn the Fleet you've configured in the UI into a GitOps repository. You end up with the same folders and files `fleetctl new` creates, filled in with your real settings, fleets, profiles, scripts, policies, reports, labels, and software. Secrets become environment variables, and a read-only `fleetctl gitops --dry-run` proves that applying the repository would change nothing.

The export only reads from Fleet. It doesn't change your Fleet instance, and it doesn't apply the repository. You do that later, through CI, after you've reviewed it.

It replaces `fleetctl generate-gitops`. [references/parity-checklist.md](references/parity-checklist.md) compares the two.

## Before you start

You need:

- A Fleet user with the global admin role.
- `fleetctl` 4.84.0 or later, ideally the same version as your Fleet server.
- Python 3.9 or later. On Windows, run it as `py -3` or `python`.
- An AI coding agent that can run commands: Claude Code, Codex, Cursor, GitHub Copilot, Gemini CLI, or Kilo Code.
- About 15 minutes, plus a few minutes per fleet to review the result.

If you're trying the skill out, point it at a test or staging Fleet first.

## Step 1: Install fleetctl and log in

1. Install `fleetctl`:

   ```bash
   npm install -g fleetctl
   ```

   To match your Fleet server's version, add it to the package name, for example `fleetctl@4.94.0`. [Other ways to install fleetctl](https://fleetdm.com/guides/fleetctl#installing-fleetctl).

2. Point `fleetctl` at your Fleet server and log in:

   ```bash
   fleetctl config set --address https://fleet.example.com
   fleetctl login
   ```

   If you sign in to Fleet with SSO or two-factor authentication, `fleetctl login` won't work. Get an API token from your account page in the Fleet UI, then run `fleetctl config set --token <token>` instead. [Learn more](https://fleetdm.com/guides/fleetctl#users-with-single-sign-on-sso-or-email-two-factor-authentication-2-fa).

3. Check that it works:

   ```bash
   fleetctl get config
   ```

## Step 2: Add the skill to your agent

The skill is this folder: `SKILL.md` plus `scripts/` and `references/`.

- Working in the Fleet repository: there's nothing to do. Claude Code finds it in `.claude/skills/`, and Codex, Cursor, Copilot, Gemini CLI, and Kilo Code find it through `.agents/skills/fleet-gitops-export/`.
- Anywhere else: download this folder:

  ```bash
  git clone --depth 1 --filter=blob:none --sparse https://github.com/fleetdm/fleet.git fleet-skill
  git -C fleet-skill sparse-checkout set .claude/skills/fleet-gitops-export
  ```

  Then copy `fleet-skill/.claude/skills/fleet-gitops-export` into `~/.claude/skills/` for Claude Code, or into `.agents/skills/` in the folder where you start Codex, Copilot, Gemini CLI, or Cursor. Some agents ask you to trust a skill the first time they load it.

## Step 3: Ask your agent to export

1. Open a terminal in the folder where the new repository should live, and start your agent.
2. Ask it, for example:

   > Export our Fleet to a GitOps repo in ./fleet-gitops

   In Claude Code, you can also type `/fleet-gitops-export ./fleet-gitops`.

3. Answer the agent's questions:
   - Is this the right Fleet? It shows the server URL `fleetctl` is logged in to. Say yes only if it's the Fleet you mean to export.
   - Organization name: it suggests the one Fleet has.
   - Folder: it has to be empty, unless you agree to the agent writing into it.

The agent creates the layout, reads everything from Fleet, and writes a plan. This takes a minute or two on most instances, and longer with hundreds of profiles or software titles.

## Step 4: Answer the review questions

Fleet doesn't record a few things, so the agent checks them with you before it writes the files. You might see:

- Which platform a script is for. Fleet runs `.sh` and `.py` scripts on macOS and Linux and doesn't record which one a script targets. The agent reads each script and asks only when it can't tell.
- Apple profiles in a mobile fleet. Fleet stores one Apple platform for macOS, iOS, and iPadOS, so the agent confirms where those profiles go.
- Fleet-maintained app scripts. Fleet supplies these apps' install and uninstall scripts and keeps them current unless someone edits them. The API doesn't say which ones were edited, so when a stored script looks like an older copy of Fleet's, the agent asks whether anyone changed it. If someone did, the export keeps it. Otherwise, Fleet keeps supplying the current one.

The agent also lists gaps: things GitOps can't represent, such as bootstrap packages, users, and API tokens. They stay as they are in Fleet.

## Step 5: Add secret values for the check (optional)

Fleet never returns some values: custom variables (`$FLEET_SECRET_...`) and tokens it shows only masked. To check the files that use them with their real values:

1. In the new repository, copy `.env.example` to `.env`.
2. Fill in the values you want checked, and leave the rest empty.

Git ignores `.env`, and the agent doesn't open it, so these values never pass through the agent. If you skip this step, the check still runs and tells you which files it couldn't check with real values.

## Step 6: Read the result

The agent validates the repository and reports:

- PASS or FAIL. PASS means every file reference resolves, the repository matches Fleet item by item, and the dry run would change nothing.
- What was exported, for each fleet.
- Gaps, and why GitOps can't carry them.
- Environment variables to set in CI. `.env.example` and the repository's README list them too.

Read through the files before you commit them.

## Step 7: Turn on GitOps in CI

The repository includes a GitHub Actions workflow and a GitLab pipeline. On GitHub, the workflow does a dry run for pull requests. It applies the repository to Fleet when you push to `main`, when you run it by hand, and every night on a schedule. Add the secrets before anything reaches `main`:

1. Create an API-only user for CI with the GitOps role (on Fleet Free, use the global admin role). The command prints the user's API token.

   ```bash
   fleetctl user create --name "GitOps" --api-only --global-role gitops
   ```

   [Learn more about API-only users](https://fleetdm.com/guides/fleetctl#create-api-only-user).

2. Create an empty, private repository on GitHub or GitLab.
3. Add these as repository secrets (GitHub) or CI/CD variables (GitLab):
   - `FLEET_URL`: your Fleet server's URL.
   - `FLEET_API_TOKEN`: the API-only user's token.
   - Every variable in `.env.example`, with its real value.
4. Push the repository to a branch other than `main`, and open a pull request. Check that the workflow's dry run passes and shows nothing you didn't expect.
5. Merge the pull request. From now on, merging to `main` applies the repository to Fleet.
6. In Fleet, go to **Settings > Integrations > Change management** and turn on **GitOps mode**. Otherwise, the next run overwrites changes made in the Fleet UI.

## Step 8: Clean up

Ask the agent to clean up, or run `python3 <skill folder>/scripts/gitops_export.py clean --dir ./fleet-gitops`. This deletes the export's work folder, which holds copies of your profiles and scripts outside the repository. Delete `.env` too if you don't need it.

## Troubleshooting

- "isn't a global admin": log in to `fleetctl` as a user with the global admin role.
- Version warning: `fleetctl` and your Fleet server are different versions. Install the matching `fleetctl` (step 1) so the dry run behaves the way CI's will.
- "can't reach Fleet" though fleetctl works in your terminal: your agent runs commands in a sandbox without network access. Allow network access for these commands.
- `unknown key "org_settings.yara_rules..."`: a known `fleetctl` bug. GitOps can't carry YARA rules until it's fixed.
- Dry-run notes about placeholders or masks: those values weren't in `.env`, so the files that use them weren't checked with real values. Add them (step 5) and ask the agent to validate again.
- Anything else: ask the agent what failed. Every step is safe to run again. [Talk to a human at Fleet](https://fleetdm.com/support) if you're stuck.

## Run the scripts without an agent

```bash
python3 scripts/gitops_export.py all --dir ~/fleet-gitops --org-name "Acme, Inc."
```

`all` runs `preflight`, `scaffold`, `export`, `place`, `validate`, and `report` in a row, and exits non-zero when validation fails. Run them one at a time to review the plan before anything is written. [SKILL.md](SKILL.md) describes each step.

## Tests

```bash
python3 -m unittest discover -s scripts/tests
```

The tests run offline. Inside the Fleet repository, they also check the templates `fleetctl new` ships, Fleet's default package scripts, and the list of deprecated GitOps keys.

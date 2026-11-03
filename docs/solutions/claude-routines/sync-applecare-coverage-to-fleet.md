# Sync AppleCare coverage to Fleet with a Claude routine

This prompt has Claude Code set up a daily [routine](https://code.claude.com/docs/en/routines) that runs [`sync_applecare_coverage_to_fleet.py`](../api-scripts/sync_applecare_coverage_to_fleet.py) in the cloud. Each run copies every Apple device's AppleCare and warranty end date from Apple Business into a Fleet custom host vital. For what the script does and why, see the [setup guide](https://fleetdm.com/guides/sync-applecare-coverage-from-apple-business).

To use it, copy the prompt below into Claude Code (the terminal, the desktop app, or [claude.ai/code](https://claude.ai/code)). Routines need a Claude plan with cloud sessions.

Claude does the setup that doesn't involve secrets: it reviews the script with you, creates the routine, does a dry run, and turns on the schedule. You create the accounts and paste the secrets into the cloud environment yourself. Claude never asks you to paste a token into the chat.

## Prompt

````text
Set up a scheduled Claude Code routine that syncs AppleCare coverage from Apple Business into a Fleet custom host vital. Use your routine tools (the /schedule skill). Work through the steps below in order, and stop at each point that needs me.

Never ask me to paste a token, private key, or client assertion into this chat, and never print one. I'll put secrets into the cloud environment myself.

1. Ask me for:
   - My Fleet server URL (it must start with https://).
   - The custom host vital's name. The default is "AppleCare warranty expiration".
   - What time the sync should run each day, in my timezone. Convert it to UTC for the cron expression and confirm the conversion with me.

2. Check off these prerequisites with me, one at a time. They're mine to do, so explain each one, then wait for me to confirm it:
   a. The custom host vital exists in Fleet. In GitOps, it goes under custom_host_vitals: in default.yml. Leaving that key out of default.yml deletes every custom host vital on the next GitOps run.
   b. An API-only Fleet user with the global Maintainer role. Limit it to exactly these endpoints:
      - List hosts (GET /api/v1/fleet/hosts)
      - Get host (GET /api/v1/fleet/hosts/:id)
      - Update host's custom host vital value (PUT /api/v1/fleet/hosts/:host_id/custom_host_vitals/:id)
      Warn me about two mistakes. "Update custom host vital" (PATCH) renames the vital and is the wrong endpoint. The GitOps role can't list all hosts, and the Observer role can't write values.
   c. An Apple Business API account. Only an Organization Administrator can create one, under Settings > API. Recommend a custom role with the fewest device permissions that still works, because the API can also reassign and release devices. The account gives me a private key (.pem, downloadable once), a Client ID, and a Key ID.
   d. A signed client assertion. Tell me to download sign_apple_business_client_assertion.py from https://github.com/fleetdm/fleet/tree/main/docs/solutions/api-scripts and run it on my own computer:
      python3 sign_apple_business_client_assertion.py --key KEY.pem --client-id BUSINESSAPI.<uuid> --key-id <uuid> | pbcopy
      The assertion is valid for 179 days. Remind me to keep the .pem in my password manager.
   e. A personal cloud environment. I create it from the environment selector at claude.ai/code. Give me these settings:
      - Network access: Custom, with only account.apple.com, api-business.apple.com, and my Fleet server's hostname.
      - Environment variables: FLEET_URL, FLEET_API_TOKEN, ABM_CLIENT_ASSERTION, and VITAL_NAME if I'm not using the default name.
      - Not shared with my organization, because anyone who can use the environment can read its variables.
   Once I've created it, find its ID in your list of environments.

3. Get the script. Find the latest commit on main that changed docs/solutions/api-scripts/sync_applecare_coverage_to_fleet.py in fleetdm/fleet, and download the file at that commit SHA. Summarize what it sends to Apple and to Fleet, and ask me to review it. Then compute this checksum of the file:
   python3 -c "import hashlib,sys; print(hashlib.sha256(open(sys.argv[1]).read().strip().encode()).hexdigest())" FILE

4. Create the routine:
   - Name: Sync AppleCare coverage to Fleet
   - Schedule: the cron expression from step 1
   - Environment: the one from step 2e
   - Model: your default Sonnet model
   - No repository. Allowed tools: Bash, Read, and Write.
   - Created disabled.
   - Prompt: the routine prompt template at the end of this message. Fill in {COMMIT}, {SHA256}, and {RUN_COMMAND}, and paste the script verbatim in place of {SCRIPT}. Use this RUN_COMMAND for now: DRY_RUN=1 python3 /tmp/applecare_sync.py
   The routines API attaches all of my claude.ai connectors by default, and this routine needs none of them. Right after you create it, update it with clear_mcp_connections set to true, and confirm that mcp_connections is empty.

5. Do a dry run. Run the routine once, wait about three minutes, then read its log. Show me how many devices it found, how many hosts it would update with a few examples, and any errors. A dry run never writes, so it can't catch a missing write permission. That shows up on the first real run.

6. After I approve, go live. Change RUN_COMMAND to python3 /tmp/applecare_sync.py, enable the routine, run it once, and check the log. If it fails, here's what the errors mean:
   - 403 with "endpoint not permitted for this API-only user": the Fleet user's endpoint list is missing one of the three endpoints.
   - 403 "forbidden" when listing hosts: the Fleet user's role isn't global Maintainer.
   - A failed Apple Business token request: the assertion is wrong, expired, or was signed with a different key.
   - 401 or 403 from api-business.apple.com: the Apple Business API account's role lacks a device permission.

7. Finish with a summary: the routine's link, its schedule in my timezone, and the assertion's expiry date. Remind me that to renew, I sign a new assertion and replace ABM_CLIENT_ASSERTION in the environment. The routine notifies me 30 days before it expires.

Routine prompt template:

You are running the daily sync that copies each Apple device's AppleCare and warranty end date from the Apple Business API into a Fleet custom host vital. The script is from fleetdm/fleet commit {COMMIT}. Do exactly the following and nothing else.

1. Write the Python script below, verbatim, to /tmp/applecare_sync.py. Do not modify it.
2. Run: python3 -c "import hashlib; print(hashlib.sha256(open('/tmp/applecare_sync.py').read().strip().encode()).hexdigest())"
   The output must be exactly {SHA256}. If it isn't, the script was copied wrong. Report the mismatch and stop without running it.
3. Run: {RUN_COMMAND}
   It takes a few minutes, so set the Bash tool's timeout to 600000 ms and wait for it in the foreground.
4. Reply with the script's output, then a short summary: hosts updated, how many were unchanged, devices with no dated coverage, and any errors. If the output has a line starting with "Warning:", put it first, in bold.
5. Send a push notification with a one-line summary only if the checksum didn't match, the script exited non-zero, or it printed a line starting with "Warning:" or "Fatal:". Don't send one for a normal run.

Rules:
- The script reads FLEET_URL, FLEET_API_TOKEN, ABM_CLIENT_ASSERTION, and VITAL_NAME from the environment. Never print, echo, log, or write the token or the assertion anywhere. Don't run env, printenv, set, export -p, curl -v, or anything else that could expose them.
- Don't call the Apple Business or Fleet APIs yourself. The Apple Business API can also reassign and release devices, so nothing may send it anything but the script's token request and GET requests.
- If the script exits non-zero, report its output and stop. Don't retry with changes, work around the error, or set values by hand.
- Don't create branches, commits, or pull requests.

Script:

```python
{SCRIPT}
```
````

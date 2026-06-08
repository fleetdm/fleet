# Secure local admin passwords with LAPS and 1Password

Managing local administrator passwords is one of those things everyone knows matters but few want to deal with. Static passwords get shared across teams, written on sticky notes, or worse—used on multiple machines. That's a security nightmare waiting to happen.

The good news? With Fleet and 1Password Connect, you can automate the entire process. Think of it as setting up password rotation once and forgetting about it—except IT can still grab credentials when needed.

## What this solves

**The problem:** Local admin accounts on macOS and Windows typically use passwords that never change. When someone leaves the team or a device is compromised, you're stuck manually resetting passwords across hundreds (or thousands) of machines.

**The solution:** This LAPS implementation automatically rotates local administrator passwords every 7 days and stores them securely in 1Password. Fleet handles the automation, and 1Password keeps everything locked down until an authorized admin needs access.

## How it works

Fleet policies monitor your devices and check whether the local admin password was rotated in the last week. When a host fails the check, Fleet automatically runs a script that:

* Generates a strong, random password
* Stores it in your 1Password vault via the Connect API
* Updates the local admin account
* Tags the credential with the device's hostname, username, and timestamp

The password never touches plaintext outside the 1Password Connect server until the script applies it directly to the local account.

## Setting it up

### What you'll need

* Fleet Premium (for secrets and automated script execution)
* A 1Password Connect server accessible from your Fleet-managed hosts
* Your Fleet GitOps repo configured

### Deploy 1Password Connect

First, set up a 1Password Connect server following the [official guide](https://developer.1password.com/docs/connect/get-started/). You'll need three pieces of information:

* Connect server URL
* API token
* Vault ID where passwords will be stored

### Configure Fleet secrets

Add these secrets in Fleet so they're available for server-side substitution in your scripts:

* `FLEET_SECRET_OP_CONNECT_HOST` – Your Connect server endpoint
* `FLEET_SECRET_OP_CONNECT_TOKEN` – API authentication token
* `FLEET_SECRET_OP_VAULT_ID` – Target vault for credentials
* `FLEET_SECRET_LAPS_ADMIN_USERNAME` - Username created/rotated (default: laps-admin)

### Add scripts and policies

Clone the [laps-1password repository](https://github.com/kc9wwh/laps-1password) and copy the scripts into your Fleet GitOps repo's `lib/` folder. Add the policy YAML files that check for recent password rotation.

Each policy uses a `run_script` automation that triggers the platform-specific LAPS script on any host that fails the 7-day check. When you push changes to your GitOps repo, Fleet picks them up automatically.

### Verify it's working

After deployment:

* Check your 1Password vault for new credential entries tagged with hostnames
* Review Fleet's policy page to confirm hosts are passing after rotation
* Test retrieval by having an authorized admin pull a credential from 1Password

## Why this approach works

**Automation reduces risk:** No more shared passwords or manual rotation schedules. Devices handle it themselves on a fixed cadence.

**Audit trail:** Every password rotation creates a new vault entry with metadata (hostname, username, timestamp). You'll know exactly when credentials changed and which device they belong to.

**Zero-trust retrieval:** Credentials stay encrypted in 1Password until an admin with proper permissions retrieves them. Even if someone gains access to a device, they won't find plaintext passwords sitting around.

**Platform flexibility:** The same workflow supports both macOS and Windows. One policy, one set of scripts, consistent security posture.

## Testing and troubleshooting

The repository includes a complete test suite you can run locally with Docker Compose, along with manual testing checklists for both macOS and Windows.

If a rotation fails, check Fleet's script execution logs and verify the device can reach your Connect server. Firewall rules and network segmentation sometimes block API calls.

Once you've got LAPS running, consider:

* Setting up alerting for consecutive rotation failures
* Adjusting the rotation interval based on your security requirements
* Documenting the credential retrieval process for your helpdesk team
* Reviewing 1Password access policies to ensure only authorized admins can view credentials

Managing local admin passwords doesn't have to be painful. With Fleet handling the automation and 1Password securing the storage, you get enterprise-grade credential management without the enterprise-grade headaches.

<meta name="articleTitle" value="Secure local admin passwords with LAPS and 1Password">
<meta name="authorFullName" value="Josh Roskos">
<meta name="authorGitHubUsername" value="kc9wwh">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-02-13">
<meta name="description" value="A guide to implementing LAPS (Local Administrator Password Solution) with Fleet and 1Password Connect">

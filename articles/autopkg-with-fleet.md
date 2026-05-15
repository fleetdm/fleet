# Use AutoPkg with Fleet

[FleetImporter](https://github.com/autopkg/fleet-recipes) is a community-maintained AutoPkg processor that connects AutoPkg workflows to Fleet's software management. It's not an official Fleet product and isn't directly supported by Fleet, but it's actively maintained in the [autopkg/fleet-recipes](https://github.com/autopkg/fleet-recipes) repo.

If you're already using AutoPkg to download and package macOS software, FleetImporter can upload packages to Fleet automatically, either by pushing directly to your Fleet server via the API (direct mode) or by opening a pull request in your Fleet GitOps repo (GitOps mode).

## Prerequisites

- AutoPkg 2.3 or later
- Python 3.9 or later
- A Fleet API token with software management permissions
- The fleet-recipes repo and its parent recipe dependencies

## Add the fleet-recipes repo

```bash
autopkg repo-add fleet-recipes
```

Fleet recipes depend on parent recipes from other AutoPkg repos. Check [PARENT_RECIPE_DEPENDENCIES.md](https://github.com/autopkg/fleet-recipes/blob/main/PARENT_RECIPE_DEPENDENCIES.md) for the full list and add those repos too.

## Create a recipe override

Recipe overrides let you customize how a recipe runs without editing the original file. This is how you switch between direct and GitOps mode, change settings like `FLEET_TEAM_ID`, enable auto-update policies, or supply custom scripts, on a per-recipe basis.

Create an override:

```bash
autopkg make-override Google/GoogleChrome.fleet.recipe.yaml
```

AutoPkg saves the override to `~/Library/AutoPkg/RecipeOverrides/`. Open it and add or change any `Input` values you want to customize, then run it the same way you'd run any recipe:

```bash
autopkg run GoogleChrome.fleet.recipe.yaml
```

AutoPkg picks up your override automatically.

## Direct mode

Direct mode is the default. FleetImporter uploads packages to your Fleet server via the API.

### Configure Fleet credentials

```bash
defaults write com.github.autopkg FLEET_API_BASE "https://fleet.example.com"
defaults write com.github.autopkg FLEET_API_TOKEN "your-fleet-api-token"
defaults write com.github.autopkg FLEET_TEAM_ID "1"
```

`FLEET_TEAM_ID` is the numeric ID of the fleet you want to assign software to.

### Run a recipe

```bash
autopkg run Google/GoogleChrome.fleet.recipe.yaml
```

FleetImporter uploads the package, extracts the app icon automatically, and creates or updates the software entry. If the software already exists, it's updated in place.

## GitOps mode

GitOps mode is for teams managing Fleet configuration as code. Instead of uploading packages directly to Fleet, FleetImporter:

1. Uploads the package to S3
2. Generates a CloudFront URL
3. Writes or updates software YAML files in your GitOps repo
4. Opens a pull request

Fleet applies the change on the next sync after you merge.

> **Why S3?** During GitOps runs, Fleet deletes any software not defined in your YAML files during sync. Hosting packages in S3 and staging changes via pull request means you control when updates go live, so a sync can't remove software before the new version is in place.

### Additional prerequisites for GitOps mode

- boto3 1.18.0 or later, installed in AutoPkg's Python environment
- An S3 bucket and a CloudFront distribution pointing at it
- AWS credentials with read/write access to the bucket
- A GitHub token with write access to your Fleet GitOps repo

### Install boto3

```bash
/Library/AutoPkg/Python3/Python.framework/Versions/Current/bin/python3 \
  -m pip install "boto3>=1.18.0"
```

If you're using `~/.aws/credentials` for AWS auth, you can skip setting `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` below.

### Configure credentials

```bash
defaults write com.github.autopkg AWS_S3_BUCKET "my-fleet-packages"
defaults write com.github.autopkg AWS_CLOUDFRONT_DOMAIN "cdn.example.com"
defaults write com.github.autopkg AWS_ACCESS_KEY_ID "your-access-key"
defaults write com.github.autopkg AWS_SECRET_ACCESS_KEY "your-secret-key"
defaults write com.github.autopkg AWS_DEFAULT_REGION "us-east-1"
defaults write com.github.autopkg FLEET_GITOPS_REPO_URL "https://github.com/org/fleet-gitops.git"
defaults write com.github.autopkg FLEET_GITOPS_GITHUB_TOKEN "your-github-token"
```

### Enable GitOps mode in your override

All fleet-recipes support both modes from a single file. Enable GitOps mode in your override:

```yaml
Input:
  GITOPS_MODE: true
```

#### Match your repo structure

By default, FleetImporter writes software YAML files to `lib/macos/software/` and the team YAML to `teams/workstations.yml`. If your repo uses different paths, set them in your override:

```yaml
Input:
  GITOPS_MODE: true
  FLEET_GITOPS_SOFTWARE_DIR: "software/macos"
  FLEET_GITOPS_TEAM_YAML_PATH: "teams/engineering.yml"
```

This lets you adopt fleet-recipes into an existing GitOps repo without reorganizing it.

### Run a recipe

```bash
autopkg run GoogleChrome.fleet.recipe.yaml
```

FleetImporter uploads the `.pkg` to S3, generates the CloudFront URL, writes a software YAML to your configured software directory, and opens a pull request. Review and merge when you're ready.

## Auto-update policies

FleetImporter can create Fleet policies that detect devices running outdated software and trigger automatic installation. This is disabled by default.

Enable it in your override:

```yaml
Input:
  automatic_update: true
```

When enabled, FleetImporter extracts the bundle identifier from the package and generates an osquery policy that finds devices where the installed version doesn't match the current one. In direct mode, it creates or updates the policy via the Fleet API. In GitOps mode, it writes a policy YAML to your repo as part of the same pull request.

Policy names follow the pattern `autopkg-auto-update-<software-title>`, for example `autopkg-auto-update-google-chrome`. Override `AUTO_UPDATE_POLICY_NAME` to use a different pattern.

For software with non-standard version detection, provide a custom query using the `%VERSION%` placeholder:

```yaml
Input:
  automatic_update: true
  auto_update_policy_query: |
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM apps
      WHERE bundle_identifier = 'com.example.MyApp'
      AND bundle_short_version != '%VERSION%'
    );
```

A few things to know:

- Policies aren't deleted automatically when you remove software. Clean them up manually.
- Existing policies with the same name are updated in place, not duplicated.
- Policy creation failures are logged as warnings and don't block the package upload.

## Custom install and uninstall scripts

Set `install_script`, `uninstall_script`, or `post_install_script` in your override. Provide the script inline:

```yaml
Input:
  uninstall_script: |
    #!/bin/bash
    rm -rf "/Applications/MyApp.app"
```

Or reference a `.sh` file in the recipe directory:

```yaml
Input:
  uninstall_script: uninstall-myapp.sh
```

Script files stay with the original recipe, so your override doesn't need to include them.

## Troubleshoot

Run with verbose output to see what FleetImporter is doing:

```bash
autopkg run -vvv GoogleChrome.fleet.recipe.yaml
```

Common issues:

- **Authentication errors**: Verify `FLEET_API_BASE` is reachable and your token has maintainer permissions for the fleet.
- **GitOps mode not enabled**: Make sure your override sets `GITOPS_MODE: true`. Without it, the recipe runs in direct mode and requires Fleet API credentials instead.
- **S3 upload failures**: Check bucket permissions and that your AWS credentials allow both upload and delete operations.
- **PR not created**: Verify your GitHub token has write access to the repo and that `FLEET_GITOPS_REPO_URL` is correct.
- **Recipe not found**: Run `autopkg list-repos` to confirm `fleet-recipes` is added, and check [PARENT_RECIPE_DEPENDENCIES.md](https://github.com/autopkg/fleet-recipes/blob/main/PARENT_RECIPE_DEPENDENCIES.md) for any missing parent repos.

## Get help

FleetImporter is a community project. For questions and issues:

- Ask in [#autopkg](https://macadmins.slack.com/archives/C056155B4) on MacAdmins Slack
- Open an issue in the [fleet-recipes repo](https://github.com/autopkg/fleet-recipes/issues)

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="authorFullName" value="Kitzy">
<meta name="publishedOn" value="2026-04-28">
<meta name="articleTitle" value="Use AutoPkg with Fleet">
<meta name="description" value="Learn how to use the community-maintained FleetImporter AutoPkg processor to automatically upload packages to Fleet.">

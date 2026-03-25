# Fleet

These files allow you to configure, patch, and secure computing devices for your organization.

Whether you're making changes by hand or spinning them up from Slack or Teams using a tool like Claude or Kilo Code _(e.g. "Make our endpoints compliant with ISO 27001" or "Fix CVE-2026-XXXX")_, your team reviews, merges, and it deploys to thousands of endpoints in seconds. This makes it straightforward to instantly rollback a change, and history is fully tracked.

You can read more about the anatomy of these files and what they do in [Fleet's documentation](https://fleetdm.com/docs/configuration/yaml-files). You can also opt to manage particular aspects of Fleet in the graphical user interface _instead_, such as software versions or labels.

> Unsure? Talk to a human at fleetdm.com/support

## How to use

1. Install fleectl. [Learn how](https://fleetdm.com/guides/fleetctl#installing-fleetctl).

2. Open your Terminal, run `fleetctl new`, and follow instructions in the output.

## Tips

The action (GitHub) or pipeline (GitLab) runs will fail until you add `FLEET_URL` and `FLEET_API_TOKEN` as [secrets (GitHub)](#github) or [CI/CD variables (GitLab)](#gitlab).

Set `FLEET_URL` to your Fleet instance's URL (ex. https://organization.fleet.com).

If you're using Fleet Free, set the API-only user's role to global admin.

### GitHub

To add GitHub secrets, see [the GitHub docs](https://docs.github.com/en/actions/security-guides/using-secrets-in-github-actions#creating-secrets-for-a-repository).

In GitHub, enable the `Apply latest configuration to Fleet` GitHub Actions workflow, and run workflow manually. Now, when anyone pushes a new commit to the default branch, the action will run and update Fleet. For pull requests, the workflow will do a dry run only.

### GitLab

To add GitLab CI/CD variables, see [the Gitlab docs](https://docs.gitlab.com/ee/ci/variables/#define-a-cicd-variable-in-the-ui).

To ensure your Fleet configuration stays up to date even when there are no new commits, set up a scheduled pipeline:
   - In your GitLab project, go to the left sidebar and navigate to **Build > Pipeline schedules**. (In some GitLab versions, this may appear as **CI/CD > Schedules**.)
   - Click **Create a new pipeline schedule** (or **Schedule a new pipeline**).
   - Fill in the form:
      - **Description**: e.g., `Daily GitOps sync`
      - **Cron timezone**: e.g., `[UTC 0] UTC`
      - **Interval pattern**: e.g., Custom: `0 6 * * *` (runs nightly at 6AM UTC)
      - **Target branch or tag**: your default branch (e.g., `main`)
   - Click **Create pipeline schedule**.

## What is Fleet?

Fleet is high-agency device management software. It is especially popular with [IT and security teams who manage lots of endpoints](https://fleetdm.com/customers).

All source code [is public](https://fleetdm.com/transparency) and the product is supported by a [company called Fleet Device Management](https://fleetdm.com/handbook/company) that enrolls millions of laptops, tablets, phones, servers, and other computing devices in 90+ countries.

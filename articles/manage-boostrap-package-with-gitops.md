# Manage bootstrap packages with GitOps


Bootstrap packages let you install custom software during device enrollment. This guide shows you how to manage them through GitOps using Fleet's API.

> **Note:** Each team needs its own bootstrap package. Teams can't share bootstrap packages.


## Prerequisites


Before you start, you'll need:

- A Fleet GitOps setup with an API-only user account
- Access to Fleet's API endpoints
- A bootstrap package ready to upload


## Upload the bootstrap package


First, upload your bootstrap package to each team that needs it.

You can use either:

- The Fleet UI to upload the package manually
- The [Create bootstrap package](https://fleetdm.com/docs/rest-api/rest-api#create-bootstrap-package) API endpoint to upload programmatically

Repeat this step for every team that needs the package.


## Get the bootstrap package token


After uploading, retrieve the unique token for each team's bootstrap package.

Use the [Get bootstrap package metadata](https://fleetdm.com/docs/rest-api/rest-api#get-bootstrap-package-metadata) API endpoint. The response includes the token you'll need for the next step.


## Configure your GitOps team file


In each team's YAML configuration file, add the `bootstrap_package` field with the download URL:
```yaml
bootstrap_package: "https://your-fleet-instance.com/api/v1/fleet/bootstrap?token=your-token-here"
```

Replace `your-fleet-instance.com` with your Fleet instance domain and `your-token-here` with the token from the previous step.


## More information


Learn more about bootstrap packages and setup experiences in the [setup experience guide](https://fleetdm.com/guides/setup-experience#bootstrap-package).

<meta name="articleTitle" value="Manage bootstrap packages with GitOps">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-01-12">
<meta name="description" value="Learn how to manage bootstrap packages across teams using Fleet's GitOps workflow and API endpoints for automated device enrollment.">
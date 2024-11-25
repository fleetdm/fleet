# Remote deployment of YARA rules

Fleetd with osquery can scan files on macOS and Linux systems using
[YARA](https://virustotal.github.io/yara/), a matching engine particularly useful for
identifying malicious patterns in binary files. The rule contents have typically been provided
either through files placed on the filesystem, or in unauthenticated HTTP endpoints such as public
GitHub repositories.

We heard concerns from customers about making the rules publically available, so in osquery 5.14
we [added the capability](https://github.com/osquery/osquery/pull/8437) for osquery to make
authenticated requests for YARA rules. Fleet is (as of this writing) the only osquery HTTP server
implementation that supports serving authenticated YARA rules.

In this guide we demonstrate how to configure the agent and server to use this more secure remote
YARA functionality. This is supported for live queries and saved queries to the
[`yara`](https://fleetdm.com/tables/yara) table.

## Configuration

Configuration is performed in 3 steps.

1) Configure agent options to enable authenticated requests to the Fleet server.
2) Configure YARA rules in Fleet.
3) Use YARA rules in queries.

### 1 - Agent options

Configure agent options to enable YARA rule request authentication in osquery and allowlist requests
to the Fleet server. This can be perfomed via the Fleet UI, GitOps, or the API. Set the agent
options as below, replacing `FLEET_SERVER_URL` with the URL of your Fleet server (eg.
`example.fleetdm.com`):

```
config:
  options:
    # other options omitted
    yara_sigurl_authenticate: true # "on" switch for using YARA rules in Fleet
  yara:
    signature_urls:
    - https://<FLEET_SERVER_URL>/api/osquery/yara/.*  # (Fleet server URL) Also required for using YARA rules in Fleet
 ```

 ### 2 - YARA rules

 Provide YARA rules to Fleet that will be served to agents. This can be performed via GitOps, or the
 API. Reference each rule file by path under the main `org_settings` configuration. In this example,
 we assume the rule files are in a `/lib/` subdirectory. This is a directory structure like the
 [Fleet GitOps recommendations](https://github.com/fleetdm/fleet-gitops).

```
org_settings:
  yara_rules:
    - path: ./lib/rule1.yar
    - path: ./lib/rule2.yar
```

Apply this configuration with `fleetctl apply` or with your GitOps CI job.

Because rules are stored as separate files in the repository, other tools like
[YARA-CI](https://yara-ci.cloud.virustotal.com/) may be used before applying the rules to Fleet.

### 3 - Use in queries

Now the provided rules may be referenced in queries utilizing the `yara` table. Rules are available at
`https://<FLEET_SERVER_URL>/api/osquery/yara/<RULE_FILENAME>`. For example:

```
SELECT * FROM yara WHERE path="/bin/ls" AND sigurl='https://example.fleetdm.com/api/osquery/yara/test1.yar'
```

This works for both live and saved queries. Each time osquery runs the query, an authenticated HTTP
request will be made to the Fleet server requesting the referenced rule(s).

## Targeting rules with teams (Fleet Premium)

It is often desirable to run different sets of YARA rules on different devices within the
organization. To achieve this, target the _queries_ to the desired team.

For example, with `rule1.yar` and `rule2.yar` configured in the `org_settings`:

1. Ensure the agent options are configured for "No team" and/or the desired teams.

2. Target queries to the appropriate team, referencing the desired rules. For example, target a
   query referencing `rule1.yar` to the "Workstations" team and a query referencing `rule2.yar` to
   the Servers team.

<meta name="authorGitHubUsername" value="zwass">
<meta name="authorFullName" value="Zach Wasserman">
<meta name="publishedOn" value="2024-12-09">
<meta name="articleTitle" value="Remote deployment of YARA rules">
<meta name="category" value="guides">
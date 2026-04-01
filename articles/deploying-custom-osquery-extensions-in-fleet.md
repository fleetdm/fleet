# Deploying custom osquery extensions in Fleet

### Links to article series:

- Part 1: Deploying custom osquery extensions in Fleet
- Part 2: [Deploying custom osquery extensions in Fleet: A step-by-step guide](https://fleetdm.com/guides/deploying-custom-osquery-extensions-in-fleet-a-step-by-step-guide)

One of the advantages of adopting open-source solutions is their extensibility. Another is the ability to customize them to meet your exact requirements. When it comes to Fleet and `osquery`, this flexibility becomes particularly powerful when deploying custom extensions across your infrastructure.

## Why custom osquery extensions matter

Custom osquery extensions allow you to gather specific telemetry data that standard `osquery` or Fleet tables might not provide. Deploying custom extensions also allows you to query the data directly using simple `SQL` syntax, rather than dealing with the hassle of setting up [Automatic Table Construction (ATC)](https://www.linkedin.com/pulse/from-data-gaps-actions-auto-table-construction-atc-fleet-houchins-l9kqc/) or parsing files directly using tables like [`file`](https://fleetdm.com/tables/file#apple), [`file_lines`](https://fleetdm.com/tables/file_lines#apple), [`parse_json`](https://fleetdm.com/tables/parse_json#apple), and so on. Custom extensions give you the power to tailor your endpoint visibility to your organization's unique needs.

## Balancing functionality and infrastructure overhead

Fleet supports deploying custom extensions through a custom [TUF (The Update Framework)](https://theupdateframework.io/) server, as documented in the [agent configuration guide](https://fleetdm.com/docs/configuration/agent-configuration). While this approach provides centralized management and security benefits, it also introduces additional infrastructure complexity.

The TUF custom extension workflow was born out of necessitiy - before Fleet had built-in custom package deployment options. 

If you're looking to minimize infrastructure management overhead, you might wonder: "What if I could install custom extensions directly on hosts and have Fleet automatically pick them up?"

The good news is: you can.

## Automating deployment with policy-based remediation

Manual deployment doesn't scale well across large fleets. Here's where Fleet's policy-based automation becomes invaluable.

### The automation strategy

- **Create a Detection Policy:** Write a policy that checks for the existence of your custom extension
- **Leverage Policy Failure:** Since the extension doesn't exist initially, hosts will fail this policy
- **Implement Automated Remediation:** Use Fleet's automated policy remediation to trigger installation

### Benefits of this approach

- **Reduced infrastructure complexity:** No need to maintain a separate TUF server
- **Familiar deployment methods:** Use existing package management and scripting
- **Automated scale:** Policy-based remediation handles deployment across your entire fleet

## Putting it into production

While TUF servers provide enterprise-grade extension management, direct deployment offers a pragmatic alternative for organizations seeking to minimize infrastructure overhead. By combining direct filesystem deployment with Fleet's policy-based automation, you can achieve scalable custom extension deployment without the complexity of additional server infrastructure.

This approach exemplifies the flexibility that makes open source solutions so powerful — giving you the freedom to implement the deployment strategy that best fits your organization's needs and constraints. [Talk to Fleet](https://fleetdm.com/contact) to learn more!

To get the step-by-step guide for devlievering custom osquery extensions with example custom extensions from Fleet's internal deployment, see the [next article](https://fleetdm.com/guides/deploying-custom-osquery-extensions-in-fleet-a-step-by-step-guide) in this series.

About the author: Allen Houchins is Head of IT & Solutions Consulting at Fleet Device Management.

<meta name="articleTitle" value="Deploying custom osquery extensions in Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-03-05">
<meta name="description" value="Learn how to deploy custom osquery extensions directly from Fleet.">

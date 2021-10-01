# Standard query library

Fleet's standard query library includes a growing collection of useful queries for organizations deploying Fleet and osquery.

In Fleet, **informational queries** are used to run live queries that return specific information about your devices on demand. Informational queries are also added to schedules which send recurring information about your devices to your log destination.

In Fleet, **policies** are queries that allow you to check which devices pass or fail your organization’s standards.

## Importing the informational queries in Fleet

#### After cloning the fleetdm/fleet repo, import informational queries using fleetctl:
```
fleetctl apply -f standard-query-library/informational.yml
```

## Importing the policies in Fleet

#### After cloning the fleetdm/fleet repo, import policies using fleetctl:
```
fleetctl apply -f standard-query-library/policies.yml
```

## Contributors

Want to add your own query?

1. Copy the following yaml and fill in each section:
```yaml
---
apiVersion: v1
kind: query
spec:
  query: Insert query here
  purpose: What is the goal of running your query? If you run this query as a live query or schedule this query, insert "Informational." If this query is used as a policy, insert "Policy."
  name: What is your query called? Please use a human readable query name.
  description: Describe your query. What information does your query reveal or what does your query check?
  platforms: What operating systems support your query? This can usually be determined by the osquery tables included in your query. Heading to the https://osquery.io/schema webpage to see which operating systems are supported by the tables you include.
  resolve: If the query's purpose is "policy", what are the steps to resolve a device that is failing? If the query's purpose is "informational", remove this section.
  contributors: Ex. zwass,mike-j-thomas
```

2. If your query's purpose is "Informational", paste your yaml at the bottom of the [`informational.yml`](./informational.yml) file. If your query's purpose is "Policy", paste your yaml at the bottom of the [`policies.yml`](./policies.yml) file.

3. Submit a pull request to the fleetdm/fleet GitHub repository.

For instructions on submitting pull requests to Fleet check out [the Committing Changes section](../../3-Contributing/4-Committing-Changes.md#committing-changes) in the Contributors documentation.

## Additional resources

Listed below are great resources that contain additional queries.

- Osquery (https://github.com/osquery/osquery/tree/master/packs)
- Palantir osquery configuration (https://github.com/palantir/osquery-configuration/tree/master/Fleet)

# Standard query library

Fleet's standard query library includes a growing collection of useful queries for organizations deploying Fleet and osquery.

In Fleet, **informational** queries are used to run live queries that return specific information about your devices on demand. Informational queries are also added to schedules which send recurring information about your devices to your log destination.

In Fleet, **policies** are queries that allow you to check which devices pass or fail your organization’s standards.


## Importing the queries in Fleet

### After cloning the fleetdm/fleet repo, import the queries using fleetctl:
```
fleetctl apply -f docs/01-Using-Fleet/standard-query-library/standard-query-library.yml
```

## Contributors

Want to add your own query?

1. Please copy the following yaml section and paste it at the bottom of the [`standard-query-library.yml`](./standard-query-library.yml) file.
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
  resolve: If the query's purpose is "Policy", what are the steps to resolve a device that is failing? If the query's purpose is "Informational", remove this section.
  contributors: Ex. zwass,mike-j-thomas
```
2. Replace each field and submit a pull request to the fleetdm/fleet GitHub repository.

For instructions on submitting pull requests to Fleet check out [the Committing Changes section](../../03-Contributing/04-Committing-Changes.md#committing-changes) in the Contributors documentation.

## Additional resources

Listed below are great resources that contain additional queries.

- Osquery (https://github.com/osquery/osquery/tree/master/packs)
- Palantir osquery configuration (https://github.com/palantir/osquery-configuration/tree/master/Fleet)

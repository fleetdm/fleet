# Standard query library

Fleet's standard query library includes a growing collection of useful queries for organizations deploying Fleet and osquery.

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
    name: What is your query called? Please use a human readable query name.
    platforms: What operating systems support your query? This can usually be determined by the osquery tables included in your query. Heading to the https://osquery.io/schema webpage to see which operating systems are supported by the tables you include.
    description: Describe your query. What does information does your query reveal?
    query: Insert query here
    purpose: What is the goal of running your query? Ex. Detection
    remediation: Are there any remediation steps to resolve the detection triggered by your query? If not, insert "N/A."
    contributors: zwass,mike-j-thomas
  ```

2. Replace each field and submit a pull request to the fleetdm/fleet GitHub repository.

3. If you want to contribute multiple queries, please open one pull request that includes all your queries.

For instructions on submitting pull requests to Fleet check out [the Committing Changes
section](../../03-Contributing/04-Committing-Changes.md#committing-changes) in the Contributors
documentation.


## Additional resources

Listed below are great resources that contain additional queries.

- Osquery (https://github.com/osquery/osquery/tree/master/packs)
- Palantir osquery configuration (https://github.com/palantir/osquery-configuration/tree/master/Fleet)

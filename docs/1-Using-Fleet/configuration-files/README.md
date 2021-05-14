# Configuration files

Entities in Fleet, such as queries, packs, labels, agent options, and enroll secrets, can be managed with configuration files in yaml syntax.

This page contains links to examples that can help you understand the configuration options for your Fleet yaml file(s).

Examples in this directory are presented in two forms:
- [`single-file-configuration.yml`](./single-file-configuration.yml) presents multiple yaml documents in one file. One file is often easier to manage than several. Group related objects into a single file whenever it makes sense.
- The `multi-file-configuration` directory presents multiple yaml documents in separate files. They are in the following structure:

```
├─ packs
├   └─ osquery-monitoring.yml
├─ agent-options.yml
├─ enroll-secrets.yml
├─ labels.yml
├─ queries.yml
```

## Using yaml files in Fleet

A Fleet configuration is defined using one or more declarative "messages" in yaml syntax. Each message can live in it's own file or multiple in one file, each separated by `---`. Each file/message contains a few required top-level keys:

- `apiVersion` - the API version of the file/request
- `spec` - the "data" of the request
- `kind ` - the type of file/object (i.e.: pack, query, config)

The file may optionally also include some `metadata` for more complex data types (i.e.: packs).

When you reason about how to manage these config files, consider following the [General Config Tips](https://kubernetes.io/docs/concepts/configuration/overview/#general-config-tips) published by the Kubernetes project. Some of the especially relevant tips are included here as well:

- When defining configurations, specify the latest stable API version.
- Configuration files should be stored in version control before being pushed to the cluster. This allows quick roll-back of a configuration if needed. It also aids with cluster re-creation and restoration if necessary.
- Don’t specify default values unnecessarily – simple and minimal configs will reduce errors.
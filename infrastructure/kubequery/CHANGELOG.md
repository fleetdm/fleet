# kubequery change log


<a name="1.1.1"></a>
## [1.1.1](https://github.com/Uptycs/kubequery/releases/tag/1.1.1)

[Git Commits](https://github.com/Uptycs/kubequery/compare/1.1.0...1.1.1)

### New Features

### Under the Hood improvements

* Upgrade to basequery 5.0.2
* Upgraded to Go 1.17

### Table Changes

### Bug Fixes

### Documentation

### Build

### Security Issues

### Packs


<a name="1.1.0"></a>
## [1.1.0](https://github.com/Uptycs/kubequery/releases/tag/1.1.0)

[Git Commits](https://github.com/Uptycs/kubequery/compare/1.0.0...1.1.0)

### New Features

* Helm chart to install kubequery
* Support for Kubernetes 1.22

### Under the Hood improvements

* Upgrade to basequery 4.9.0
* Upgraded to client go version 0.22

### Table Changes

* k8s 1.22 caused few table [schemas changes](https://github.com/Uptycs/kubequery/commit/a70e9a42f6f85ca1a0ebd23575590c73562fab83#diff-79f5d80ee02a931b2bf12fd018b6edeb447abd58e1fb85ae155ae932ec29ad9d):
  * kubernetes_stateful_sets
  * kubernetes_jobs
  * kubernetes_persistent_volume_claims
  * kubernetes_services

### Bug Fixes

* Check container status before iterating over contents. [Issue 16](https://github.com/Uptycs/kubequery/issues/16)

### Documentation

* Added helm related details in README.md

### Build

### Security Issues

### Packs


<a name="1.0.0"></a>
## [1.0.0](https://github.com/Uptycs/kubequery/releases/tag/1.0.0)

[Git Commits](https://github.com/Uptycs/kubequery/compare/0.3.0...1.0.0)

### New Features

* New `kubequeryi` command line to easily invoke shell
* Easy to use with [query-tls](https://github.com/Uptycs/query-tls)

### Under the Hood improvements

* Upgrade to basequery 4.8.0
* Switch to light weight busybox docker image
* Simple NodeJS based integration test

### Table Changes

* Added `cluster_name` and `cluster_uid` to tables missing those columns
* Break up `resources` in `*_containers` tables to `resource_limits` and `resource_requests`
* Added new table `kubernetes_component_statuses`
* Removed table `kubernetes_storage_capacities`

### Bug Fixes

### Documentation

### Build

* Upgrade to Go 1.16

### Security Issues

### Packs

* Added default query pack for all kubernetes tables


<a name="0.3.0"></a>
## [0.3.0](https://github.com/Uptycs/kubequery/releases/tag/0.3.0)

[Git Commits](https://github.com/Uptycs/kubequery/compare/0.2.0...0.3.0)

### New Features

### Under the Hood improvements

* Upgrade to basequery 4.7.0

### Table Changes

### Bug Fixes

### Documentation

* Validate the installation was successful [PR-12](https://github.com/Uptycs/kubequery/pull/12)

### Build

### Security Issues

### Packs


<a name="0.2.0"></a>
## [0.2.0](https://github.com/Uptycs/kubequery/releases/tag/0.2.0)

[Git Commits](https://github.com/Uptycs/kubequery/compare/0.1.0...0.2.0)

### New Features

* Added `kubernetes_events` table.

### Under the Hood improvements

* Switch to [basequery](https://github.com/Uptycs/basequery). This is stripped download version of Osquery with support for extension events and other features.

### Table Changes

* kubernetes_events

### Bug Fixes

### Documentation

* Validate the installation was successful [PR-12](https://github.com/Uptycs/kubequery/pull/12)

### Build

### Security Issues

### Packs

# Fleet [![CircleCI](https://circleci.com/gh/fleetdm/fleet/tree/master.svg?style=svg)](https://circleci.com/gh/fleetdm/fleet/tree/master) [![Go Report Card](https://goreportcard.com/badge/github.com/fleetdm/fleet)](https://goreportcard.com/report/github.com/fleetdm/fleet)

Fleet is the most widely used open-source osquery Fleet manager. Deploying osquery with Fleet enables live queries, and effective management of osquery infrastructure.

Documentation for Fleet can be found on [GitHub](./docs/README.md).

**Fleet Dashboard**
![Screenshot of dashboard](./assets/images/dashboard-screenshot.png)

**Live Queries**
![Screenshot of live query interface](./assets/images/query-screenshot.png)

**Scheduled Query/Pack Editor**
![Screenshot of pack editor](./assets/images/pack-screenshot.png)

## Using Fleet

#### The CLI

If you're interested in learning about the `fleetctl` CLI and flexible osquery deployment file format, see the [CLI Documentation](./docs/cli/README.md).

#### Deploying Osquery and Fleet

Resources for deploying osquery to hosts, deploying the Fleet server, installing Fleet's infrastructure dependencies, etc. can all be found in the [Infrastructure Documentation](./docs/infrastructure/README.md).

#### Accessing The Fleet API

If you are interested in accessing the Fleet REST API in order to programmatically interact with your osquery installation, please see the [API Documentation](./docs/api/README.md).

#### The Web Dashboard

Information about using the Kolide web dashboard can be found in the [Dashboard Documentation](./docs/dashboard/README.md).

## Developing Fleet

#### Development Documentation

If you're interested in interacting with the Kolide source code, you will find information on modifying and building the code in the [Development Documentation](./docs/development/README.md).

If you have any questions, please create a [GitHub Issue](https://github.com/kolide/fleet/issues/new).

## Community

#### Chat

Please join us in the #kolide channel on [Osquery Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/).

#### Community Projects

Below are some projects created by Kolide community members. Please submit a pull request if you'd like your project featured.

- [davidrecordon/terraform-aws-kolide-fleet](https://github.com/davidrecordon/terraform-aws-kolide-fleet) - Deploy Fleet into AWS using Terraform.
- [deeso/fleet-deployment](https://github.com/deeso/fleet-deployment) - Install Fleet on a Ubuntu box.
- [gjyoung1974/kolide-fleet-chart](https://github.com/gjyoung1974/kolide-fleet-chart) - Kubernetes Helm chart for deploying Fleet.

# Kolide Fleet [![CircleCI](https://circleci.com/gh/kolide/fleet/tree/master.svg?style=svg)](https://circleci.com/gh/kolide/fleet/tree/master)

### Effective Endpoint Security. At Any Scale.

Kolide Fleet is a state of the art host monitoring platform tailored for security experts. Leveraging Facebook's battle-tested osquery project, Fleet delivers fast answers to big questions. To learn more about Fleet, visit [https://kolide.com/fleet](https://kolide.com/fleet).

Documentation for Fleet can be found on [GitHub](./docs/README.md).

[![Kolide](./assets/images/rube.png)](https://kolide.com/fleet)

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

Please join us in the #kolide channel on [Osquery Slack](https://osquery-slack.herokuapp.com/).

#### Community Projects

Below are some projects created by Kolide community members. Please submit a pull request if you'd like your project featured.

- [davidrecordon/terraform-aws-kolide-fleet](https://github.com/davidrecordon/terraform-aws-kolide-fleet) - Deploy Fleet into AWS using Terraform.
- [deeso/fleet-deployment](https://github.com/deeso/fleet-deployment) - Install Fleet on a Ubuntu box.
- [gjyoung1974/kolide-fleet-chart](https://github.com/gjyoung1974/kolide-fleet-chart) - Kubernetes Helm chart for deploying Fleet.

## Kolide Cloud

Looking for the quickest way to try out osquery on your fleet? Not sure which queries to run? Don't want to manage your own data pipeline?

Try our [osquery SaaS platform](https://kolide.com/?utm_source=oss&utm_medium=readme&utm_campaign=fleet) providing insights, alerting, fleet management and user-driven security tools. We also support advanced aggregation of osquery results for power users. Get started with your 30-day free trial [today](https://kolide.com/signup?utm_source=oss&utm_medium=readme&utm_campaign=fleet).

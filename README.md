:tada: Announcing the transition of Fleet to a new independent entity :tada:

Please check out [the blog post](https://medium.com/fleetdm/a-new-fleet-d4096c7de978) to understand what is happening with Fleet and our commitment to improving the product.  To upgrade from Fleet ≤3.2.0, just grab the latest release from this repository (it'll work out of the box).

# Fleet [![CircleCI](https://circleci.com/gh/fleetdm/fleet/tree/master.svg?style=svg)](https://circleci.com/gh/fleetdm/fleet/tree/master) [![Go Report Card](https://goreportcard.com/badge/github.com/fleetdm/fleet)](https://goreportcard.com/report/github.com/fleetdm/fleet)

Fleet is the most widely used open source osquery manager.  Deploying osquery with Fleet enables programmable live queries, streaming logs, and effective management of osquery across 50,000+ servers, containers, and laptops.  It's especially useful for talking to multiple devices at the same time.

Fleet is a Go app. You can run it on your own hardware or deploy it in any cloud.

Documentation for Fleet can be found on [GitHub](./docs/README.md).

![banner-fleet-cloud-city](https://user-images.githubusercontent.com/618009/98254443-eaf21100-1f41-11eb-9e2c-63a0545601f3.jpg)

<img alt="Screenshot of query editor" src="https://user-images.githubusercontent.com/618009/101847266-769a2700-3b18-11eb-9109-7f1320ed5c45.png"/>


<!-- todo: update other screenshots
**Fleet Dashboard**
![Screenshot of dashboard](./assets/images/dashboard-screenshot.png)

**Live Queries**
![Screenshot of live query interface](./assets/images/query-screenshot.png)

**Scheduled Query/Pack Editor**
![Screenshot of pack editor](./assets/images/pack-screenshot.png)
-->

## Using Fleet

#### The CLI

If you're interested in learning about the `fleetctl` CLI and flexible osquery deployment file format, see the [CLI Documentation](./docs/cli/README.md).

#### Deploying osquery and Fleet

Resources for deploying osquery to hosts, deploying the Fleet server, installing Fleet's infrastructure dependencies, etc. can all be found in the [Infrastructure Documentation](./docs/infrastructure/README.md).

#### Accessing The Fleet API

If you are interested in accessing the Fleet REST API in order to programmatically interact with your osquery installation, please see the [API Documentation](./docs/api/README.md).

#### The Web Dashboard

Information about using the web dashboard can be found in the [Dashboard Documentation](./docs/dashboard/README.md).

## Developing Fleet

Organizations large and small use osquery with Fleet every day to stay secure and compliant. That’s good news, since it means there are lots of other developers and security practitioners talking about Fleet, dreaming up features, and contributing patches. Let’s stop reinventing the wheel and build the future of device management together.

#### Development Documentation

If you're interested in interacting with the Fleet source code, you will find information on modifying and building the code in the [Development Documentation](./docs/development/README.md).

If you have any questions, please create a [GitHub Issue](https://github.com/fleetdm/fleet/issues/new).

## Community

#### Chat

Please join us in the #fleet channel on [osquery Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/).

#### Community Projects

Below are some projects created by Fleet community members. Please submit a pull request if you'd like your project featured.

- [Kolide](https://kolide.com) is a cloud-hosted, user-driven security SaaS application.  To be clear: Kolide ≠ Fleet.  Kolide is well-executed, a great commercial tool, and they offer a 30-day free trial.
- [davidrecordon/terraform-aws-kolide-fleet](https://github.com/davidrecordon/terraform-aws-kolide-fleet) - Deploy Fleet into AWS using Terraform.
- [deeso/fleet-deployment](https://github.com/deeso/fleet-deployment) - Install Fleet on a Ubuntu box.
- [gjyoung1974/kolide-fleet-chart](https://github.com/gjyoung1974/kolide-fleet-chart) - Kubernetes Helm chart for deploying Fleet.


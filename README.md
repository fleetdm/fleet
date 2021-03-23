<h1><img width="200" alt="Fleet logo, landscape, dark text, transparent background" src="https://user-images.githubusercontent.com/78363703/112149147-06746480-8c22-11eb-8893-031ffa99539b.png"></h1>

#### [Website](https://fleetdm.com/)  &nbsp;  [News](http://twitter.com/fleetctl) &nbsp; [Report a bug](https://github.com/fleetdm/fleet/issues/new)

[![Run Tests](https://github.com/fleetdm/fleet/actions/workflows/test.yml/badge.svg)](https://github.com/fleetdm/fleet/actions/workflows/test.yml) &nbsp; [![Go Report Card](https://goreportcard.com/badge/github.com/fleetdm/fleet)](https://goreportcard.com/report/github.com/fleetdm/fleet) &nbsp; [![Twitter Follow](https://img.shields.io/twitter/follow/fleetctl.svg?style=social&maxAge=3600)](https://twitter.com/fleetctl)

Fleet is the most widely used open source osquery manager.  Deploying osquery with Fleet enables programmable live queries, streaming logs, and effective management of osquery across 50,000+ servers, containers, and laptops.  It's especially useful for talking to multiple devices at the same time.


## Try Fleet

#### With [Node.js](https://nodejs.org/en/download/) and [Docker](https://docs.docker.com/get-docker/) installed:

```bash
# Install the Fleet command-line tool
npm install -g fleetctl
# Run a local demo of the Fleet server
sudo fleetctl preview
```

The Fleet UI is now available at http://localhost:1337.

#### Your first query
Ready to run your first query?  Target some of your sample hosts and try it out:
<img width="800" alt="Screenshot of query editor" src="https://user-images.githubusercontent.com/618009/111853677-099de680-88ea-11eb-90bb-f5cd787f1f15.png"/>

#### Using real devices
For convenience, the demo includes a few simulated Linux hosts.  To query a real device, [install the osquery agent](https://github.com/fleetdm/orbit).

## Team
Fleet is [independently backed](https://linkedin.com/company/fleetdm) and actively maintained with the help of many amazing [contributors](https://github.com/fleetdm/fleet/graphs/contributors).

> **:tada: Announcing the transition of Fleet to a new independent entity :tada:**
> 
> Please check out [the blog post](https://medium.com/fleetdm/a-new-fleet-d4096c7de978) to understand what is happening with Fleet and our commitment to improving the product.  To upgrade from Fleet ≤3.2.0, just grab the latest release from this repository (it'll work out of the box).

## Documentation

Documentation for Fleet can be found [here on GitHub](./docs/README.md).

<!-- TODO: "#### Contributing" as one-liner with link to best jumping off point in docs -->
<!-- TODO: "#### Production deployment" as one-liner with link to best jumping off point in docs -->

## Community

#### Chat

Please join us in the #fleet channel on [osquery Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/).

#### Community projects

Below are some projects created by Fleet community members. Please submit a pull request if you'd like your project featured.

- [Kolide Cloud ("K2")](https://kolide.com) is a cloud-hosted, user-driven security SaaS application.  To be clear: Kolide ≠ Fleet.
- [davidrecordon/terraform-aws-kolide-fleet](https://github.com/davidrecordon/terraform-aws-kolide-fleet) - Deploy Fleet into AWS using Terraform.
- [deeso/fleet-deployment](https://github.com/deeso/fleet-deployment) - Install Fleet on a Ubuntu box.
- [gjyoung1974/kolide-fleet-chart](https://github.com/gjyoung1974/kolide-fleet-chart) - Kubernetes Helm chart for deploying Fleet.

<a href="https://fleetdm.com"><img alt="Banner featuring a futuristic cloud city with the Fleet logo" src="https://user-images.githubusercontent.com/618009/98254443-eaf21100-1f41-11eb-9e2c-63a0545601f3.jpg"/></a>

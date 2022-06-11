<h1><img width="200" alt="Fleet logo, landscape, dark text, transparent background" src="https://user-images.githubusercontent.com/618009/103300491-9197e280-49c4-11eb-8677-6b41027be800.png"></h1>

#### [Website](https://fleetdm.com/)  &nbsp;  [News](http://twitter.com/fleetctl) &nbsp; [Report a bug](https://github.com/fleetdm/fleet/issues/new)

[![Run Tests](https://github.com/fleetdm/fleet/actions/workflows/test.yml/badge.svg)](https://github.com/fleetdm/fleet/actions/workflows/test.yml) &nbsp; [![Go Report Card](https://goreportcard.com/badge/github.com/fleetdm/fleet)](https://goreportcard.com/report/github.com/fleetdm/fleet) &nbsp; [![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/5537/badge)](https://bestpractices.coreinfrastructure.org/projects/5537) &nbsp; [![Twitter Follow](https://img.shields.io/twitter/follow/fleetctl.svg?style=social&maxAge=3600)](https://twitter.com/fleetctl) &nbsp; 

Fleet is the most widely used open source osquery manager.  Deploying osquery with Fleet enables programmable live queries, streaming logs, and effective management of osquery across 100,000+ servers, containers, and laptops.  It's especially useful for talking to multiple devices at the same time.


## Try Fleet

#### With [Node.js](https://nodejs.org/en/download/) and [Docker](https://docs.docker.com/get-docker/) installed:

```bash
# Install the Fleet command-line tool
sudo npm install -g fleetctl
# Run a local demo of the Fleet server
sudo fleetctl preview
```

> Windows users can omit `sudo`, and should run the command in `Cmd`/`PowerShell` as administrators.

The Fleet UI is now available at http://localhost:1337.

#### Now what?

Check out the [Ask questions about your devices tutorial](./docs/Using-Fleet/Learn-how-to-use-Fleet.md#how-to-ask-questions-about-your-devices) to learn where to see your devices in Fleet, how to add Fleet's standard query library, and how to ask questions about your devices by running queries.

## Team
Fleet is [independently backed](https://linkedin.com/company/fleetdm) and actively maintained with the help of many amazing [contributors](https://github.com/fleetdm/fleet/graphs/contributors).

> 📖 In keeping with our value of openness, Fleet Device Management's company handbook is public and open source.  You can read about the [history of Fleet and osquery](https://fleetdm.com/handbook/company#history) and our commitment to improving the product.
> To upgrade from Fleet ≤3.2.0, just follow the upgrading steps for the latest release from this repository (it'll work out of the box).

## Documentation

Documentation for Fleet can be found [here](https://fleetdm.com/docs).

<!-- TODO: "#### Contributing" as one-liner with link to best jumping off point in docs -->
<!-- TODO: "#### Production deployment" as one-liner with link to best jumping off point in docs -->

## Community

#### Chat

Please join us in the #fleet channel on [osquery Slack](https://fleetdm.com/slack).

#### Contributing

Contributions are welcome, whether you answer questions on Slack/GitHub/StackOverflow/Twitter, improve the documentation or website, write a tutorial, give a talk, start a local osquery meetup, troubleshoot reported issues, or [submit a patch](https://github.com/fleetdm/fleet/blob/main/CONTRIBUTING.md).  The Fleet code of conduct is [on GitHub](https://github.com/fleetdm/fleet/blob/main/CODE_OF_CONDUCT.md).

<a href="https://fleetdm.com"><img alt="Banner featuring a futuristic cloud city with the Fleet logo" src="https://user-images.githubusercontent.com/618009/98254443-eaf21100-1f41-11eb-9e2c-63a0545601f3.jpg"/></a>

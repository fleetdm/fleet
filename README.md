<h1><a href="https://fleetdm.com"><img width="200" alt="Fleet logo, landscape, dark text, transparent background" src="https://user-images.githubusercontent.com/618009/103300491-9197e280-49c4-11eb-8677-6b41027be800.png"></a></h1>

#### [News](https://fleetdm.com/announcements) &nbsp; · &nbsp; [Report a bug](https://github.com/fleetdm/fleet/issues/new) &nbsp; · &nbsp; [Handbook](https://fleetdm.com/handbook/company) &nbsp; · &nbsp; [Why open source?](https://fleetdm.com/handbook/company/why-this-way#why-open-source) &nbsp; · &nbsp; [Art](https://fleetdm.com/logos)

Open-source platform for IT and security teams with thousands of computers.  Designed for APIs, GitOps, webhooks, YAML, and humans.

<a href="https://fleetdm.com/logos"><img src="https://github.com/fleetdm/fleet/assets/618009/f835ec29-1cb9-49ba-a0f3-395ffd9d5c9f" alt="A glass city in the clouds"/></a>


## What's it for?
Organizations like Fastly and Gusto use Fleet for vulnerability reporting, detection engineering, device management (MDM), device health monitoring, posture-based access control, managing unused software licenses, and more.

#### Explore data
To see what kind of data you can use Fleet to gather, check out the [table reference documentation](https://fleetdm.com/tables).

#### Out-of-the-box policies
Fleet includes out-of-the box support for all [CIS benchmarks for macOS and Windows](https://fleetdm.com/docs/using-fleet/cis-benchmarks), as well as many [simpler queries](https://fleetdm.com/queries).

Take as much or as little as you need for your organization.

#### Supported platforms
Here are the platforms Fleet currently supports:

- Linux (all distros)
- macOS
- Windows
- Chromebooks
- Amazon Web Services (AWS)
- Google Cloud (GCP)
- Azure (Microsoft cloud)
- Data centers
- Containers (kube, etc)
- Linux-based IoT devices

## Lighter than air
Fleet is lightweight and modular.  You can use it for security without using it for MDM, and vice versa.  You can turn off features you are not using.

#### Openness
Fleet is dedicated to flexibility, accessibility, and clarity.  We think [everyone can contribute](https://fleetdm.com/handbook/company#openness) and that tools should be as easy as possible for everyone to understand.

#### Good neighbors
Fleet has no ambition to replace all of your other tools.  (Though it might replace some, if you want it to.)  Ready-to-use, enterprise-friendly integrations exist for Snowflake, Splunk, GitHub Actions, Vanta, Elastic Jira, Zendesk, and more.

Fleet plays well with Munki, Chef, Puppet, and Ansible, as well as with security tools like Crowdstrike and SentinelOne.  For example, you can use the free version of Fleet to quickly report on what hosts are _actually_ running your EDR agent.

#### Free as in free
The free version of Fleet will [always be free](https://fleetdm.com/pricing).  Fleet is [independently backed](https://linkedin.com/company/fleetdm) and actively maintained with the help of many amazing [contributors](https://github.com/fleetdm/fleet/graphs/contributors).

#### Longevity
The [company behind Fleet](https://fleetdm.com/handbook/company) is founded (and majority-owned) by [true believers in open source](https://fleetdm.com/handbook/company/why-this-way#why-open-source).  The company's business model is influenced by GitLab (NYSE: GTLB), with great investors, happy customers, and the capacity to become profitable at any time.

In keeping with Fleet's value of openness, [Fleet Device Management's company handbook](https://fleetdm.com/handbook/company) is public and open source.  You can read about the [history of Fleet and osquery](https://fleetdm.com/handbook/company#history) and our commitment to improving the product.

<!-- > To upgrade from Fleet ≤3.2.0, just follow the upgrading steps for the earliest subsequent major release from this repository (it'll work out of the box until the release of Fleet 5.0). -->


## Is it any good?
Fleet is used in production by IT and security teams with thousands of laptops and servers.  Many deployments support tens of thousands of hosts, and a few large organizations manage deployments as large as 400,000+ hosts.



## Chat
Please join us in [MacAdmins Slack](https://www.macadmins.org/) or in [osquery Slack](https://fleetdm.com/slack).

The Fleet community is full of [kind and helpful people](https://fleetdm.com/handbook/company#empathy).  Whether or not you are a paying customer, if you need help, just ask.


## Contributing &nbsp; [![Run Tests](https://github.com/fleetdm/fleet/actions/workflows/test.yml/badge.svg)](https://github.com/fleetdm/fleet/actions/workflows/test.yml) &nbsp; [![Go Report Card](https://goreportcard.com/badge/github.com/fleetdm/fleet)](https://goreportcard.com/report/github.com/fleetdm/fleet) &nbsp; [![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/5537/badge)](https://bestpractices.coreinfrastructure.org/projects/5537) &nbsp; [![Twitter Follow](https://img.shields.io/twitter/follow/fleetctl.svg?style=social&maxAge=3600)](https://twitter.com/fleetctl) &nbsp; 

The landscape of cybersecurity and IT is too complex.  Let's open it up.

Contributions are welcome, whether you answer questions on [Slack](https://fleetdm.com/slack) / [GitHub](https://github.com/fleetdm/fleet/issues) / [StackOverflow](https://stackoverflow.com/search?q=osquery) / [LinkedIn](https://linkedin.com/company/fleetdm) / [Twitter](https://twitter.com/fleetctl), improve the documentation or [website](./website), write a tutorial, give a talk at a conference or local meetup, give an [interview on a podcast](https://fleetdm.com/podcasts), troubleshoot reported issues, or [submit a patch](https://fleetdm.com/docs/contributing/contributing).  The Fleet code of conduct is [on GitHub](https://github.com/fleetdm/fleet/blob/main/CODE_OF_CONDUCT.md).

<!-- - Great contributions are motivated by real-world use cases or learning.
- Some of the most valuable contributions might not touch any code at all.
- Small, iterative, simple (boring) changes are the easiest to merge. -->

## What's next?
To see what Fleet can do, head over to [fleetdm.com](https://fleetdm.com) and try it out for yourself, grab time with one of the maintainers to discuss, or visit the docs and roll it out to your organization.

#### Production deployment
Fleet is simple enough to [spin up for yourself](https://fleetdm.com/docs/get-started/tutorials-and-guides).  Or you can have us [host it for you](https://fleetdm.com/pricing).  Premium features are [available](https://fleetdm.com/pricing) either way.

#### Documentation
Complete documentation for Fleet can be found at [https://fleetdm.com/docs](https://fleetdm.com/docs).


## License
The free version of Fleet is available under the MIT license.  The commercial license is also designed to allow contributions to paid features for users whose employment agreements allow them to contribute to open source projects.  (See LICENSE.md for details.)

> Fleet is built on [osquery](https://github.com/osquery/osquery), [nanoMDM](https://github.com/micromdm/nanomdm), [Nudge](https://github.com/macadmins/nudge), and [swiftDialog](https://github.com/swiftDialog/swiftDialog).

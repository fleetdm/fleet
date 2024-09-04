# Anatomy
This page details the core concepts you need to know to use Fleet.

## Fleet UI
Fleet UI is the GUI (graphical user interface) used to control Fleet. [Docs](https://fleetdm.com/docs/using-fleet/fleet-ui).

## Fleetctl
Fleetctl (pronouced “fleet control”) is a CLI (command line interface) tool for managing Fleet from the command line. [Docs](https://fleetdm.com/docs/using-fleet/fleetctl-cli).

## Fleetd
Fleetd is a bundle of agents provided by Fleet to gather information about your devices. Fleetd includes [osquery](https://www.osquery.io/), [Orbit](https://github.com/fleetdm/fleet/blob/main/orbit/README.md), Fleet Desktop, and the [Fleetd Chrome extension](https://github.com/fleetdm/fleet/blob/main/ee/fleetd-chrome/README.md).

## Osquery
Osquery is an open-source tool for gathering information about the state of any device that the osquery agent has been installed on. [Learn more](https://www.osquery.io/).

## Orbit
Orbit is an osquery version and configuration manager, built by Fleet.

## Fleet Desktop
Fleet Desktop is a menu bar icon that gives end users visibility into the security and status of their machine. [Docs](https://fleetdm.com/docs/using-fleet/fleet-desktop).

## Fleetd Chrome extension
The Fleetd Chrome extension enrolls ChromeOS devices in Fleet. [Docs](https://github.com/fleetdm/fleet/blob/main/ee/fleetd-chrome/README.md).

## Host
A host is a computer, server, or other endpoint. Fleet gathers information from Fleet's agent (fleetd) installed on each of your hosts. [Docs](https://fleetdm.com/docs/using-fleet/adding-hosts).

## Team
A team is a group of hosts. Use teams to segment your hosts into groups that reflect your organization's IT and security policies. [Docs](https://fleetdm.com/docs/using-fleet/teams).

## Query
A query in Fleet refers to an osquery query. Osquery uses basic SQL commands to request data from hosts. Use queries to manage, monitor, and identify threats on your devices. [Docs](https://fleetdm.com/docs/using-fleet/fleet-ui).

## Policy
A policy is a specific “yes” or “no” query. Use policies to manage security compliance in your
organization. Learn more [here](https://fleetdm.com/securing/what-are-fleet-policies).

## Host vitals
Host vitals are the hard-coded queries Fleet uses to populate device details.

## Software library
An inventory of each host’s installed software, including information about detected vulnerabilities (CVEs).

<meta name="pageOrderInSection" value="200">

# Anatomy
This page details the core concepts you need to know to use Fleet.

## Fleet UI
Fleet UI is the GUI (graphical user interface) used to control Fleet. [Learn more](https://youtu.be/1VNvg3_drow?si=SWyQSEQMoHUYDZ8C).

## Fleetctl
Fleetctl (pronouced “fleet control”) is a CLI (command line interface) tool for managing Fleet from the command line. [Docs](https://fleetdm.com/docs/using-fleet/fleetctl-cli).

## Fleetd
Fleetd is a bundle of agents provided by Fleet to gather information about your devices. Fleetd includes:
- **Osquery:** an open-source tool for gathering information about the state of any device that the osquery agent has been installed on. [Learn more](https://www.osquery.io/).
- **Orbit:** an osquery version and configuration manager, built by Fleet. [Learn more](https://github.com/fleetdm/fleet/blob/main/orbit/README.md)
- **Fleetd Chrome extension:** enrolls ChromeOS devices in Fleet. [Docs](https://github.com/fleetdm/fleet/blob/main/ee/fleetd-chrome/README.md).

## Fleet Desktop
Fleet Desktop is a menu bar icon that gives end users visibility into the security and status of their machine. [Docs](https://fleetdm.com/docs/using-fleet/fleet-desktop).

## Host
A host is a computer, server, or other endpoint. Fleet gathers information from Fleet's agent (fleetd) installed on each of your hosts. [Docs](https://fleetdm.com/docs/using-fleet/adding-hosts).

## Team

A team is a group of hosts. Organize hosts into teams to apply queries, policies, scripts, and other configurations tailored to their specific risk and compliance requirements. [Read the guide](https://fleetdm.com/guides/teams).

## Query
A query in Fleet refers to an osquery query. Osquery uses basic SQL commands to request data from hosts. Use queries to manage, monitor, and identify threats on your devices. [Docs](https://fleetdm.com/docs/using-fleet/fleet-ui).

## Policy
A policy is a specific “yes” or “no” query. Use policies to manage security compliance in your
organization. [Read the guide](https://fleetdm.com/securing/what-are-fleet-policies).

## Host vitals
Fleet's built-in queries for collecting and storing important device information.

## Software
Software in Fleet refers to the following:
- **Software library:** a collection of Fleet-maintained apps, VPP, and custom install packages that can be installed on your hosts. [See available software](https://fleetdm.com/app-library).
- **Software inventory** an inventory of each host’s installed software, including information about detected vulnerabilities (CVEs). 

<meta name="pageOrderInSection" value="200">

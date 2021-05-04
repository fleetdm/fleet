<img width="200" alt="Fleet logo, landscape, dark text, transparent background" src="https://user-images.githubusercontent.com/618009/103300491-9197e280-49c4-11eb-8677-6b41027be800.png">

# Orbit osquery

Orbit is an [osquery](https://github.com/osquery/osquery) runtime and autoupdater. With Orbit, it's easy to deploy osquery, manage configurations, and stay up to date. Orbit eases the deployment of osquery connected with a [Fleet server](https://github.com/fleetdm/fleet), and is a (near) drop-in replacement for osquery in a variety of deployment scenarios.

Orbit is the recommended agent for Fleet. But Orbit can be used with or without Fleet, and Fleet can be used with or without Orbit.

## Try Orbit

#### With [`fleetctl preview` already running](https://github.com/fleetdm/fleet#try-fleet) and [Go](https://golang.org/doc/install) installed:

```bash
# From within the top-level directory of this repo…
# Generate a macOS installer pointed at your local Fleet
go run ./cmd/package --type=pkg --fleet-url=localhost:8412 --insecure --enroll-secret=YOUR_FLEET_ENROLL_SECRET_HERE
```

> With fleetctl preview running, you can find your Fleet enroll secret by selecting the "Add new host" button on the Hosts page in the Fleet UI.

An installer configured to point at your Fleet instance has now been generated.

Now run that installer (double click, on a Mac) to enroll your own computer as a host in Fleet. Refresh after several seconds (≈30s), and you should now see your local computer as a new host in Fleet.

## Bugs

To report a bug or request a feature, [click here](https://github.com/fleetdm/fleet/issues).

## Capabilities

| Capability                           | Status |
| ------------------------------------ | ------ |
| Secure autoupdate for osquery        | ✅     |
| Secure autoupdate for Orbit          | ✅     |
| Configurable update channels         | ✅     |
| Full osquery flag customization      | ✅     |
| Package tooling for macOS `.pkg`     | ✅     |
| Package tooling for Linux `.deb`     | ✅     |
| Package tooling for Linux `.rpm`     | ✅     |
| Package tooling for Windows `.msi`   | ✅     |
| Manage/update osquery extensions     | 🔜     |
| Manage cgroups for Linux performance | 🔜     |

## Usage

[The up-to-date documentation for Orbit is maintained in the Fleet repository.](https://github.com/fleetdm/fleet/blob/master/docs/2-Orbit-osquery/README.md)

General information and flag documentation can be accessed by running `orbit --help`.

## Community

#### Chat

Please join us in the #fleet channel on [osquery Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/).

<a href="https://fleetdm.com"><img alt="Banner featuring a futuristic cloud city with the Fleet logo" src="https://user-images.githubusercontent.com/618009/98254443-eaf21100-1f41-11eb-9e2c-63a0545601f3.jpg"/></a>

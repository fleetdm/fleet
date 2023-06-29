# Orbit development and release strategy

Goal: Define strategy that Orbit developers must follow when introducing new features.

## Why do we need a strategy?

Orbit and Fleet use a different release strategy. Orbit components are updated via "automatic updates" by continuously polling https://tuf.fleetctl.com/ for new versions, whereas on-premises Fleet servers are updated manually by administrators.
For this reason we need a good release strategy to not break on-premise deployments when we release new versions of Orbit components.

## Must Rule

"New Orbit versions always supports communication+operation with an older Fleet server"

> Why is it a must?

As mentioned before, Orbit uses an auto-update mechanism, whereas Fleet does not.
We don't want to break on-premise Fleet deployments, and we don't want to force Orbit users to update their server everytime we push a new Orbit update to Fleet's TUF server.

## Nice to have

Nice to have, but not a must: "New Fleet server version to support old version of Orbit"

> Why is it not a must?

This allows some flexibility when developing new features in Orbit and Fleet.

## Release process

1. Orbit components (Orbit itself, Fleet Desktop and osqueryd) must be released to FleetDM's TUF before new Fleet server releases are available in Github.
2. When the new Fleet server version doesn't support older Orbit versions (see [Nice to have](#nice-to-have)), the release notes must document their minimum supported Orbit version. This is for users that use Orbit with auto-updates disabled or they pin to a specific channel. These users would need to first update Orbit in their devices and then proceed to upgrade Fleet server.

<meta name="pageOrderInSection" value="1200">
<meta name="description" value="A page outlining the strategy that developers must follow when introducing new feature to Fleetd.">
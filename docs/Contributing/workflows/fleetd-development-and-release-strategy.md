# Fleetd development and release strategy

Goal: Define strategy that Fleetd developers must follow when introducing new features.

## Why do we need a strategy?

Fleetd and Fleet use different release strategies. Fleetd components are updated via "automatic updates" by continuously polling https://tuf.fleetctl.com/ for new versions, whereas on-premises Fleet servers are updated manually by administrators.
For this reason we need a good release strategy to not break on-premise deployments when we release new versions of Fleetd components.

## Must rule

"New Fleetd versions always support communication + operation with older Fleet servers."

> Why is it a must?

As mentioned before, Fleetd uses an auto-update mechanism, whereas Fleet does not.
We don't want to break on-premise Fleet deployments, and we don't want to force Fleetd users to update their servers every time we push a new Fleetd update to Fleet's TUF server.

## Nice to have

Nice to have, but not a must: "New Fleet server versions support old versions of Fleetd."

> Why is it not a must?

This allows some flexibility when developing new features in Fleetd and Fleet.

## Release process

1. Fleetd components (Orbit, Fleet Desktop and osqueryd) must be released to FleetDM's TUF before new Fleet server releases are available in Github.
2. When the new Fleet server version doesn't support older Fleetd versions (see [Nice to have](#nice-to-have)), the release notes must document their minimum supported Fleetd version. This is for users that use Fleetd with auto-updates disabled or they pin to a specific channel. These users would need to first update Fleetd on their devices and then proceed to upgrade Fleet server.

<meta name="pageOrderInSection" value="1200">
<meta name="description" value="An outline of the strategy that developers must follow when introducing new features to fleetd.">
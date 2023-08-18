# High level architecture

- [Overview](#overview)
- [Components](#components)

## Overview

Add text

## Components

```mermaid
graph LR;
    tuf["<a href=https://theupdateframework.io/>TUF</a> file server<br>(default: <a href=https://tuf.fleetctl.com>tuf.fleetctl.com</a>)"];
    fleet_server[Fleet<br>Server];

    subgraph Agent
        orbit[orbit];
        desktop[Fleet Desktop<br>Tray App];
        osqueryd[osqueryd];

        desktop_browser[Fleet Desktop<br> from Browser];
    end

    subgraph Customer Cloud
        orbit[orbit];
        desktop[Fleet Desktop<br>Tray App];
        osqueryd[osqueryd];

        desktop_browser[Fleet Desktop<br> from Browser];
    end

    subgraph FleetDM Cloud
        orbit[orbit];
        desktop[Fleet Desktop<br>Tray App];
        osqueryd[osqueryd];

        desktop_browser[Fleet Desktop<br> from Browser];
    end

    orbit -- "Fleet Orbit API (TLS)" --> fleet_server;
    desktop -- "Fleet Desktop API (TLS)" --> fleet_server;
    osqueryd -- "osquery<br>remote API (TLS)" --> fleet_server;
    desktop_browser -- "My Device API (TLS)" --> fleet_server;

    orbit -- "Auto Update (TLS)" --> tuf;
```


## Capabilities

| Capability                           | Status |
| ------------------------------------ | ------ |

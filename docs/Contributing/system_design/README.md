# Fleet DM System Architecture

## Static running Architecture

1. [Server](./server.md)
2. [Database](./database.md)
3. [Cache](./cache.md)
4. [Fleetd](./fleetd.md)
5. [Mobile Hosts](./mobile-host.md)
6. [TUF](./TUF.md)
7. [UI](./UI.md)

```mermaid
graph TD
    subgraph Client Devices
        A[Client Devices]
        D[Fleetd Agents]
        G[Fleet Desktop UI]
    end

    A -->|Enroll| B[Fleet Server]
    B -->|Store Data| C[Database]
    B -->|cache Data| R[Cache]
    B -->|Query| D
    D -->|Collect Data| A
    B -->|API Requests| E[Fleet Server UI]
    E -->|Display Data| F[IT Admins]
    B -->|Device API Requests| G[Fleet Server UI]
    G -->|Display Host Data| H[End User]
    I[TUF server] -->|Pull updates| D
```

In more detail

## Main System Components

```mermaid
graph LR;
    
    subgraph Development
        fleet_release_owner[Fleet Release<br>Owner];
    end

    subgraph Agent
        orbit[orbit];
        desktop[Fleet Desktop<br>Tray App];
        osqueryd[osqueryd];
        desktop_browser[Fleet Desktop<br> from Browser];
    end

    subgraph Customer Cloud
        fleet_server[Fleet<br>Server];
        db[DB];
        redis[Redis<br>Live queries' results <br>go here];
        prometheus[Prometheus Server];
    end

    subgraph FleetDM Cloud
        tuf["<a href=https://theupdateframework.io/>TUF</a> file server<br>(legacy: <a href=https://tuf.fleetctl.com>tuf.fleetctl.com</a>)<br>(default: <a href=https://updates.fleetdm.com>updates.fleetdm.com</a>)"];
        datadog[DataDog metrics]
        heroku[Usage Analytics<br>Heroku]
        log[Send logs to optional<br> external location]
    end

    subgraph Customer Admin
        frontend[API user UI or other]
    end


    fleet_release_owner -- "Release Process" --> tuf;

    orbit -- "Fleet Orbit API (TLS)" --> fleet_server;
    orbit -- "Auto Update (TLS)" --> tuf;
    desktop -- "Fleet Desktop API (TLS)" --> fleet_server;
    osqueryd -- "osquery<br>remote API (TLS)" --> fleet_server;
    desktop_browser -- "My Device API (TLS)" --> fleet_server;

    heroku -- "Metrics from all customers" --> datadog;

    fleet_server <== "Read/Write" ==> db;
    fleet_server <== "Read/Write" ==> redis;
    redis <==> db;

    prometheus ==> fleet_server;
    fleet_server -- "metrics" --> heroku;
    fleet_server -- "queries results" --> log;

    frontend <== "API" ==> fleet_server;

```

## Workflows

### Configuring the server
#### UI / Env var
#### Gitops

### Enrolling hosts
#### Osquery only
#### Fleetd package
#### Automatic enrollment
#### BYOD MDM

## Features

### MDM
### Orchestration
### Software
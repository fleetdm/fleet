# High level architecture

- [Overview](#overview)
- [Main System Components](#main-system-components)

## Overview

Add text

## Main System Components

```mermaid
graph LR;
    fleet_release_owner[Fleet Release<br>Owner];

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
        tuf["<a href=https://theupdateframework.io/>TUF</a> file server<br>(default: <a href=https://tuf.fleetctl.com>tuf.fleetctl.com</a>)"];
        datadog[DataDog metrics]
        heroku[Usage Analytics<br>Heroku]
        log[Optional Log Location<br>Store logs here]
    end

    subgraph Customer Admin
        frontend[frontend code]
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



## The path of Live Query

### IT person initiates the query
```mermaid
graph LR;
    it_person[IT person<br>Starts a live query];
    api[API Client Frontend or Fleetctl];

    subgraph Cloud
        server(Server);
        dbredis[DB / Redis];
    end

    it_person --> api;
    api --> it_person;

    api <-- "1 - Initiate Campaign. Get ID" --> server;
    api <-- "2 - Register to notifications with ID" --> server;
    api <-- "WEB SOCKET" --> server;
    server <-- Notifications --> dbredis;

```

### Agent returns results
```mermaid
graph LR;
    osquery[osquery agent];

    subgraph Cloud
        server(Server);
        dbredis[DB / Redis];
    end

    osquery --> server;
    server --> osquery;

    osquery <-- 1 - ask for queries<br>2 - return results --> server;


```

# High level architecture

- [Overview](#overview)
- [Main System Components](#main-system-components)

## Overview

Add text

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
        tuf["<a href=https://theupdateframework.io/>TUF</a> file server<br>(default: <a href=https://tuf.fleetctl.com>tuf.fleetctl.com</a>)"];
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



## The path of Live Query

### 1 - Fleet User initiates the query
```mermaid
graph LR;
    it_person[Fleet User<br>Starts a live query];
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

### 2 - Agent returns results
```mermaid
graph LR;
    osquery[osquery agent];

    subgraph Cloud
        server(Server);
        dbredis[DB / Redis];
    end

    osquery -- 1 ask for queries --> server;
    osquery -- 2 return results --> server;

    server <-- 1 return queries if found --> dbredis;
    server -- 2 put results in Redis --> dbredis;

```

## The path of a scheduled Query

### 1 - Fleet User initiates the query
```mermaid
graph LR;
    it_person[Fleet User<br>Creates a scheduled<br>for a team / global];
    api[API Client Frontend or Fleetctl];

    subgraph Cloud
        server(Server);
        db[DB];
    end

    it_person --> api;
    api --> server;
    server -- Query stored in DB--> db;
```
### 2 - Agent gets config file (with the scheduled query)
```mermaid
graph LR;
    agent[Osquery Agent];

    subgraph Cloud
        server(Server);
        db[DB];
    end

    agent -- request download config file --> server;
    agent <-- teams and global cfg are merged --> server;
    server -- ask for cfg file--> db;
```

### 3 - Agent returns results to be (optionally) logged
```mermaid
graph LR;
    agent[Osquery Agent<br>Runs query and sends results];

    subgraph Cloud
        server(Server);
        log[Optional External Log<br>e.g. S3];
    end

    agent --> server;
    server --> log;
```


## Agent  config options
1 - Config TLS refresh 
(Typical period 10 secs) OSQuery pulls down a config file that includes instructions for Scheduled Queries. 
If both GLOBAL and TEAM is configured, there will be a config merge done on the Server side. 

2 - Logger TLS
(Typical period10 secs) Frequency of sending the results. (different than the frequency of running the queries)
To be improved: Currently the config file gets downloaded every time even if no change was done.

3 - Distributed (Typical interval 10 sec)
(Typical period10 secs) OSQuery asks for any Live query to run.


## Vulnerability dashboard
Typically hosted on our Heroku servers.
Could be hosted on customer servers.
Uses the Fleet server Token to access specific APIs that give information about vulnerability
status.

### Vuln dashboard hosted by FleetDM
```mermaid
graph LR;
    frontend[Frontend on web browser];

    subgraph Customer Cloud
        fleetServer(Fleet Server);
    end

    subgraph Heroku Cloud
        vulnServer(Vuln Web Server);
    end

    frontend --> vulnServer;
    vulnServer --> fleetServer;
```

<meta name="pageOrderInSection" value="1201">
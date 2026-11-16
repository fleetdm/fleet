# Check the warehouse before you buy another laptop

*Spare devices sit on a shelf, powered off and out of view. Here's how Fleet's IT team brings warehouse status into Fleet, and why it's worth doing.*

## Key takeaways

- **Spare devices are an asset inventory blind spot.** A laptop in storage stops checking in, so your inventory knows it exists but not whether it's ready to ship, in repair, or headed for disposal.
- **A warehouse service tracks the part your tools can't see.** Services like Retriever handle returns, storage, redeployment, and disposal, and record each device's status as it moves.
- **Status is most useful where your team already works.** Copying it into a Fleet custom host vital puts it on the host's page, next to everything else Fleet knows about the device.
- **A label turns a status into a shopping list.** One host vitals label shows every device that's ready to ship, so "Do we have a spare?" no longer needs a vendor portal login.
- **The sync runs with minimal access.** A daily job reads Retriever's API and updates Fleet through an API-only user limited to three endpoints.
- **Reuse only happens if people know what's on the shelf.** Fleet's handbook shows employees how to find available devices and what each status means.

<a purpose="cta-button" href="https://fleetdm.com/guides/sync-warehouse-status-from-retriever">Set up the Retriever sync</a>

When someone leaves a company, their laptop usually goes back to IT. When someone joins, IT needs a laptop to send them. In between, the device sits in a closet, an office, or a third-party warehouse, and that's where many asset inventories go quiet.

Fleet's IT team wanted a quick answer to "Do we have a spare that fits?" without logging in to another portal or keeping a spreadsheet in sync. The answer turned out to be putting the warehouse's own status data into Fleet.

## Why spare devices fall out of view

Device management tools learn about a device from the device. Fleet's agent reports a host's hardware, software, and settings on a regular schedule while the host is online. A laptop in storage is powered off, so its record in Fleet stops at its last check-in. Fleet still has the serial number, the model, and the last user, but it can't tell you whether the device has been wiped, repaired, or set aside for disposal.

IT asset management (ITAM) is supposed to cover that gap. In practice, warehouse status often lives in a spreadsheet or a vendor portal that only a few people check, and it drifts from reality.

## What a warehouse service adds

[Retriever](https://helloretriever.com) handles the physical side. It retrieves devices from departing employees, stores them, and redeploys or disposes of them on request. As a device moves, Retriever updates its status, like Return Initiated, Device Received, Ready For Deployment, Deployed, or Disposed. Retriever exposes those statuses through its API.

That makes Retriever the system of record for where a spare device is and what condition it's in. What's missing is a link between that record and everything else you know about the device.

## Put warehouse status next to everything else

Fleet's [custom host vitals](https://fleetdm.com/guides/custom-host-vitals) let you add your own field to every host and set its value through the API. Fleet's IT team added one called **Warehouse status** and fills it from Retriever every day. The status now shows on each host's details page, next to its hardware, operating system, and last-seen details.

A [host vitals label](https://fleetdm.com/guides/managing-labels-in-fleet) turns one status into a list. Fleet's **Warehouse: Ready for deployment** label shows every device Retriever has ready to ship. When an equipment request comes in, IT, or the person asking, can check that label before anyone buys new hardware.

Because it's a custom host vital, the status can also feed other Fleet features, like scripts, configuration profiles, and host name templates.

## How Fleet's IT team runs the sync

Fleet doesn't have a built-in Retriever integration. The sync is a short script that uses both public APIs. Once a day, it reads every device from Retriever's warehouse API, matches each serial number to a Fleet host, and updates the host's status only when it changed. Fleet's IT team runs it as a Claude Code routine, and the same script works under cron or any other scheduler.

The script's access is kept small on both sides:

- **In Fleet**, it uses an API-only user with the maintainer role, limited to three endpoints: list hosts, get a host, and set a host's custom host vital value. Fleet rejects any other request from that token, so it can't run scripts or change settings.
- **In Retriever**, it only sends read requests. Retriever API keys can also submit device return orders, so the script never sends anything else.

The custom host vital and the label are defined in Fleet's own GitOps configuration, in [`default.yml`](https://github.com/fleetdm/fleet/blob/main/it-and-security/default.yml) and a [label file](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/all/labels/warehouse-status.yml). Changes to them go through a pull request like any other change.

## Tell people how to use it

A status nobody knows about doesn't save any money. Fleet's handbook has a [section on checking warehouse inventory](https://fleetdm.com/handbook/it#check-warehouse-inventory) that tells employees how to find devices that are ready to ship and explains each status. The IT team's own process for new equipment starts with the same check, before buying anything.

## Use what you already have

Every device on the shelf is a purchase you might not need to make. Getting its status into the tool your team already uses every day is what turns a stored laptop back into a working one.

## See it live

- [Sync warehouse status from Retriever to Fleet](https://fleetdm.com/guides/sync-warehouse-status-from-retriever): the step-by-step guide, including the script and the minimum API access.
- **Get a demo:** [Talk to Fleet](https://fleetdm.com/contact) about tracking your devices from purchase to disposal.

<meta name="articleTitle" value="Check the warehouse before you buy another laptop">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-05">
<meta name="category" value="articles">
<meta name="description" value="How Fleet's IT team brings Retriever warehouse status into Fleet, so spare devices get reused before anyone buys new hardware.">

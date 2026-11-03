# The power of collaboration: an admin-rights audit trail for every Mac

*Three open-source projects, three sets of maintainers, one week. Here's how a new Privileges release turned into a table you can query across your whole fleet.*

## Key takeaways

- **Open-source fixes ripple fast.** A feature in one project doesn't stay there for long. SAP Privileges 2.6.0, a new table in the Mac Admins osquery extension, and support in Fleet's agent all landed within about a week.
- **Admin rights on macOS now have a real audit trail.** Privileges 2.6.0 uses Apple's Endpoint Security framework to record every admin grant and removal, including which process made the change.
- **Short-lived, self-service elevation is the middle ground.** Permanent admin rights weaken security, and constant approval requests slow engineers down. Privileges lets people elevate when they need to and drops the rights again automatically.
- **One command turns it on, but plan the deployment.** Enabling the system extension takes a single CLI call. In a real rollout, Full Disk Access and the app's settings belong in configuration profiles, not manual clicks.
- **The data is a query away, on every Mac at once.** The new `privileges_events` table exposes the same audit trail through Fleet's agent, so you can ask thousands of Macs who gained admin, and how, instead of running a CLI on each one.
- **Standard tables already cover install state.** Whether Privileges is installed, which version, and whether its system extension is enabled all come from the `apps` and `system_extensions` tables. Only the audit events needed something new.

<a purpose="cta-button" href="https://fleetdm.com/tables/privileges_events">See the privileges_events table</a>

Open-source projects rarely stand alone. Developers move between them, building on each other's work, and I like watching how a fix or feature in one place doesn't take long to ripple into the next.

Here's a recent example from the Mac Admins community, and why it matters if you manage admin rights on macOS.

## osquery: the shared foundation

The story starts with [osquery](https://github.com/osquery/osquery), the foundation of open-source device telemetry since Facebook released it in 2014. It's also fundamental to how Fleet works. Fleet's agent, fleetd, is built on osquery, so you ask questions of every device using SQL. Fleet developers actively contribute to core osquery.

Around osquery, a larger community has grown. One long-running effort from the Mac Admins world is [macadmins/osquery-extension](https://github.com/macadmins/osquery-extension). Collaborators from industry and education build the tables many admins need, and Fleet bundles many of them directly into fleetd, so admins get them without extra setup. If you'd rather ship your own, we cover [deploying custom osquery extensions in Fleet](https://fleetdm.com/guides/deploying-custom-osquery-extensions-in-fleet-a-step-by-step-guide) in a separate guide.

## SAP Privileges gets a real audit trail

At Fleet, we often point customers to the open-source [Privileges](https://github.com/SAP/macOS-enterprise-privileges) app for macOS, and it's in Fleet's software catalog as a Fleet-maintained app. People work as standard users day to day, request admin rights only when they need them, and Privileges removes those rights again automatically.

Privileges has been around since 2018, built by Mac admins and developers at SAP and released as open source. It has grown into a widely used tool, with more than 2,000 stars on GitHub.

One challenge for IT is handling requests for admin rights, especially from engineering teams who need more control over their devices to build and test software. Permanent admin rights weaken security, but making engineers ask for elevation every time slows them down. Privileges aims for a middle ground: short-lived, self-service elevation instead of either extreme.

Over past releases, Privileges has picked up a set of features aimed squarely at that kind of enterprise use:

- Exclude specific admin accounts from automatic privilege revocation at login
- Use mutual TLS (mTLS) for webhook notifications
- Point the app's help button to a custom URL, like an internal support article
- Pass custom data, such as a machine name or serial number, to a webhook
- Run a script or binary automatically when privileges change

It has also gradually gotten better at noticing when admin rights change outside the app. Since version 2.3.0, a background daemon called PrivilegesWatcher has detected admin changes made through some other route and logged a generic message. Version 2.5.0 added a system extension that locks Privileges down against tampering.

The newest release, Privileges 2.6.0, takes that a step further with a full audit trail sourced from Apple's Endpoint Security framework. Run `PrivilegesCLI --history --json` and you get a history of every `ADMIN_ADD` and `ADMIN_REMOVE` event on the device, including which process made the change: Privileges itself, the Users & Groups pane, or a script running quietly in the background. Instead of knowing only that something changed, you now know what changed, when, and how.

SAP publishes its own [open source manifesto](https://www.sap.com/about/company/innovation/open-source.html), committing to build in the open and hand tools back to the community. Privileges, and this whole chain of events, is a good example of that in practice.

### Try it yourself

Turning on Endpoint Security support takes one command. If the system extension isn't already running:

```
❯ PrivilegesCLI -e on
System extension enabled

Please grant the Privileges system extension
full disk access, otherwise it will not work.
```

In a real deployment, granting Full Disk Access through an MDM profile is a requirement of its own, not something to approve by hand on every Mac.

Privileges ships a sample settings profile inside its app bundle, at `/Applications/Privileges.app/Contents/Resources/Privileges.mobileconfig`, covering many of its features. It's a good starting point for pushing settings out to every Mac at once instead of configuring devices one at a time.

Once the Endpoint Security feature is enabled, `-h` pulls the history, and `-T` scopes it to today:

```
❯ PrivilegesCLI -h -T
2026-08-01T08:41:21Z   PrivilegesDaemon   SAPCorp: User henry now has standard user privileges (privileges expired)
2026-08-01T08:41:52Z   PrivilegesDaemon   SAPCorp: User henry now has administrator privileges
```

Add `-j` and you get the same data as structured JSON, which is what the new table parses.

## One new table, useful across every fleet

That's useful on a single Mac. It's more useful across thousands of them, which is what a new addition to macadmins/osquery-extension provides: a `privileges_events` table that exposes the same audit trail through osquery, so the data can be queried across an entire fleet instead of one device at a time.

Query it instead of the CLI, and the same events come back as rows:

```
SELECT event_type, principal, subject, signing_id, pid, is_platform_binary, process_start_time
FROM privileges_events
WHERE last = '1d';
```

```
        event_type = ADMIN_REMOVE
         principal = henry
           subject = user
        signing_id = corp.sap.privileges.daemon
               pid = 97406
is_platform_binary = 0
process_start_time = 2026-09-10T15:44:14Z

        event_type = ADMIN_ADD
         principal = henry
           subject = user
        signing_id = corp.sap.privileges.daemon
               pid = 8948
is_platform_binary = 0
process_start_time = 2026-09-10T15:45:39Z

        event_type = ADMIN_REMOVE
         principal = ad-man
           subject = user
        signing_id = com.apple.Users-Groups-Settings.extension
               pid = 11896
is_platform_binary = 1
process_start_time = 2026-09-10T15:46:01Z
```

Same device, same events, but now it's a query away instead of a command you have to run locally on the Mac in question. The third row is the interesting one: an admin change made through System Settings, not through Privileges, and the table tells you so.

The install-state side of things (whether Privileges is installed, which version, and whether the system extension is enabled) turns out not to need a custom table at all. Standard osquery tables already cover it:

```
SELECT a.bundle_short_version AS version,
       EXISTS (SELECT 1 FROM system_extensions
               WHERE identifier = 'corp.sap.privileges.extension'
                 AND state = 'activated_enabled') AS extension_enabled
FROM apps a WHERE a.path = '/Applications/Privileges.app';
```

So `privileges_events` is the only new table, since audit events sourced from Endpoint Security process data aren't available anywhere else in osquery.

Fleet already ships several tables from this extension, and this one is now bundled into fleetd too. Fleet admins can start querying it as soon as their hosts pick up the agent update.

## Three projects, one week

What I like most about this case is the timeline. The Privileges update, the new table in the Mac Admins extension, and Fleet's support for it all came together within about a week. Three separate projects, three separate teams of maintainers, one shared goal.

I had fun helping coordinate this one, and I'm glad to keep working alongside so many people in this community.

## See it live

- Read the [privileges_events table reference](https://fleetdm.com/tables/privileges_events) for every column and a scheduled-query example.
- Deploy Privileges from Fleet's [software catalog](https://fleetdm.com/software-catalog) as a Fleet-maintained app.
- [Get a demo](https://fleetdm.com/contact) to see live queries across macOS, Windows, and Linux.

<meta name="articleTitle" value="The power of collaboration: an admin-rights audit trail for every Mac">
<meta name="authorFullName" value="Henry Stamerjohann">
<meta name="authorGitHubUsername" value="headmin">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-09-16">
<meta name="description" value="How SAP Privileges 2.6.0, a new Mac Admins osquery table, and Fleet's agent came together in a week to give you a queryable admin-rights audit trail.">

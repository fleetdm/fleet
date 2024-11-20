# FAQ

## Can you host Fleet for me?

Fleet offers managed cloud hosting for [Fleet Premium](https://fleetdm.com/pricing) customers with large deployments.

> While organizations of all kinds use Fleet, from Fortune 500 companies to school districts to hobbyists, today we are only currently able to provide fully-managed hosting for deployments larger than 300 hosts.  (Instead, you can [buy a license](https://fleetdm.com/customers/register) and self-host Fleet Premium with support.)

Fleet is simple enough to [spin up for yourself](https://fleetdm.com/docs/deploy/introduction).  Premium features are [available](https://fleetdm.com/pricing) either way.

## What is the easiest way to deploy Fleet?

Take a look at [Deploying Fleet](https://fleetdm.com/docs/deploy/deploy-fleet) for guides on Render or deploy for scale with AWS Terraform. You can also deploy Fleet anywhere you want.

You can enroll servers and laptops using a simple installer or automatically deliver Fleet's agent (fleetd) using your existing tools, such as Chef, Terraform, Munki/autopkg, Ansible, Puppet, Jamf, Intune, etc. As of [Fleet v4.53.0](https://fleetdm.com/releases/fleet-4.53.0), when Fleet's MDM is enabled for MacOS, fleetd is installed as part of the bootstrap.

By default, Fleet keeps fleetd up to date automatically.  For self-managed instances, Fleet provides a [Simple upgrade process](https://fleetdm.com/docs/deploy/upgrading-fleet#upgrading-fleet).

## What options do I have for access control?  What about auditing admin activity?

Fleet lets you define [role-based access](https://fleetdm.com/docs/using-fleet/manage-access#manage-access) controls, the ability to assign different admins for different groups of hosts, and rolling deployments. 
Fleet has built-in [audit logging](https://fleetdm.com/docs/rest-api/rest-api#activities) (accessed through API or streamed to a data lake.) 

In addition, you can do GitOps because you can control your Fleet instance through a git repo, allowing you to use your standard CI/CD and approval process.  This also tracks the history of changes as commits.

## Does Fleet include pre-built queries?

Fleet comes with a [built-in query library](https://fleetdm.com/queries) for reporting device health and also includes over 400 optional [built-in CIS policies](https://fleetdm.com/docs/using-fleet/cis-benchmarks) for Mac and Windows.

You can easily write queries yourself with query auto-complete, as well as import query packs for HID to detect IOCs using Yara or other intrusion detection mechanisms from the community or other vendors. Or, you can import policies to monitor for high-impact vulnerabilities such as a particular TPM chip; for example, a large vehicle manufacturer uses Fleet to do this.

Customers can build on these built-in policies to monitor ongoing compliance with regulatory standards like NIST, PCI, ISO, SOC, and HIPAA.

## Has anyone stress-tested Fleet? How many hosts can the Fleet server handle?

Fleet is used in production deployments with hundreds of thousands of hosts and has been stress-tested to 150,000 online and 400,000 total enrolled hosts.

It’s standard deployment practice to have multiple Fleet servers behind a load balancer.  However, the MySQL database is typically the performance bottleneck and a single Fleet server can handle tens of thousands of hosts.

## Will Fleet slow down my servers?  What about my employee laptops?

Unlike legacy systems, Fleet gives you complete control over how frequent and labor-intensive the scanning is.

When you collect data with Fleet, the [performance impact](https://fleetdm.com/releases/fleet-4.5.0) is automatically reported.  You can analyze CPU, memory, and network usage or just compare whether a query's performance impact is “minimal,” “considerable,” or “excessive.”  You can easily compare the average performance of a scan across all systems or troubleshoot performance on an individual host.  If one of your queries gets too rowdy on a particular host, Fleet will [temporarily disable it](https://fleetdm.com/docs/using-fleet/osquery-process).

You can test changes on a small subset of hosts first, then roll them out to the rest of your organization.

## What browsers does Fleet supported?

Fleet supports the latest, stable releases of all major browsers and platforms.

We test each browser on Windows whenever possible, because our engineering team primarily uses macOS.

**Note:** This information also applies to [fleetdm.com](https://www.fleetdm.com).

### Desktop

- Chrome
- Firefox
- Edge
- Safari (macOS only)

### Mobile

- Mobile Safari on iOS
- Mobile Chrome on Android

### Note
> - Mobile web is not yet supported in the Fleet product.
> - The Fleet user interface [may not be fully supported](https://github.com/fleetdm/fleet/issues/969) in Google Chrome when the browser is running on ChromeOS.

## What host operating systems does Fleet support?

Fleet supports the following operating system versions on hosts. 

| OS         | Supported version(s)                    |
| :--------- | :-------------------------------------- |
| macOS      | 13+ (Ventura)                           |
| iOS/iPadOS | 17+                                     |
| Windows    | Pro and Enterprise 10+, Server 2012+    |
| Linux      | CentOS 7.1+, Ubuntu 20.04+, Fedora 38+, Debian 11+ |
| ChromeOS   | 112.0.5615.134+                        |

While Fleet may still function partially or fully with OS versions older than those above, Fleet does not actively test against unsupported versions and does not pursue bugs on them.

## Some notes on compatibility

### Tables
Not all osquery tables are available for every OS. Please check out the [osquery schema](https://fleetdm.com/tables) for detailed information. 

If a table is not available for your host, Fleet will generally handle things behind the scenes for you.

### Linux

Fleet Desktop is supported on Ubuntu and Fedora.

Fedora requires a [gnome extension](https://extensions.gnome.org/extension/615/appindicator-support/) and Google Chrome for Fleet Desktop.

Fleet's default (un)install scripts use `apt-get` for Debian-based distributions, and `dnf` for Red Hat-based distributions. To install packages on CentOS versions prior to 8, either add `dnf` or edit install and uninstall scripts to use the `yum` or `rpm` command.

On Ubuntu, Fleet Desktop currently supports Xorg as X11 server, Wayland is currently not supported. Ubuntu 24.04 comes with Wayland enabled by default. To use X11 instead of Wayland you can set `WaylandEnable=false` in `/etc/gdm3/custom.conf` and reboot.

The `fleetctl package` command is not supported on DISA-STIG distribution.

## Is Fleet MIT licensed?

Different portions of the Fleet software are licensed differently, as noted in the [LICENSE](https://github.com/fleetdm/fleet/blob/main/LICENSE) file. The majority of Fleet is MIT licensed. Paid features require a license key.

## What is your commitment to open source stewardship?

- When a feature is free and open source we won't move that feature to a paid tier. Features might be removed from the open source codebase in other cases, for example when combining features from multiple tiers into one new feature.

- The majority of new capabilities added to Fleet will benefit all users, not just customers.

- We won't introduce features into the open source codebase with a fixed delay; if a feature is planned to land in both it will be released simultaneously in both.

- We will always release and open source all tests that we have for any open source feature.

- The free version of Fleet is enterprise ready.

- The open source codebase will not contain any artificial limits on the number of hosts, users, size, or performance.

- The majority of new features contributed by Fleet Device Management Inc will be open source.

- The product will be available for download without leaving an email address or logging in.

- We will always allow you to benchmark the performance of Fleet. (Fleet also [load tests the platform before every release](https://fleetdm.com/handbook/engineering#rituals), with increasingly ambitious targets. The scale of real time reporting supported by Fleet has increased 5,000% since 2019. Today, Fleet deployments support 500,000 devices, and counting. The company is committed to driving this number to 1M+, and beyond.)

## How do I contact Fleet for support?

A lot of questions can be answered [in the documentation](https://fleetdm.com/docs) or [guides](https://fleetdm.com/guides).

To get help from the community, visit https://fleetdm.com/support.

If your organization has Fleet Premium, you can [access professional support](https://fleetdm.com/customers/login) with a guaranteed response time.

## What if we choose not to renew?

If you opt not to renew Fleet Premium, you can continue using only the free capabilities of Fleet (same code base, just unconfigure the license key.)

## Can we buy a license to access premium features with reduced support for a reduced cost?

We aren’t able to sell licenses and support separately.

## Do you offer pricing for unmanaged hosts? What about ephemeral hosts which may scale up or down?

For now, the number of hosts is the maximum cap of hosts enrolled at any given time. Umanaged hosts ("Pending" MDM status in Fleet) are not included in the enrolled hosts count.

## When run locally, what resources does the Fleet app typically consume on an individual instance, and when run in HA, at high volume? And how is latency on an individual instance vs clustered deployment?

Like any modern application, Fleet scales horizontally. The biggest potential bottleneck for Fleet is the number of hosts being monitored, so that's where we've devoted the most attention when testing. The largest number of hosts we've had a customer ask about was 350,000, for all of the production servers and employee laptops of a publicly traded company.

## Where's the data stored?

Since Fleet is self-managed, some metadata is stored wherever it is deployed (e.g. Amazon, Azure, Google, your own data center, hybrid cloud, anywhere). That's done using a MySQL database, but the bulk of the data flows directly into a tool like Splunk or ElasticSearch. You can send that information to any of Fleet's supported log destinations.

## Can I fork Fleet's source code and build upon it myself to create my own features?

Potentially! Fleet is open core with a [source code license](https://github.com/fleetdm/fleet/blob/main/LICENSE) similar to GitLab's.

Anyone is free to contribute to the free or paid features of the project. We are always interested to hear feedback, and we are happy to take pull requests and ideas upstream any time we can. 


## Can I buy support or services separate from Fleet Premium?

The only way we are able to partner as a business to provide support and build new open source and paid features is through customers purchasing Fleet Premium.

## How can I uninstall fleetd?
To uninstall Fleet's agent (fleetd), follow the instructions [here](https://fleetdm.com/guides/how-to-uninstall-fleetd).

<!--
Mike T: In 2023 we made the decision to comment out the following questions because the FAQs had become a dumping ground for miscellaneous content that wasn't quite reference docs and wasn't quite committed learning docs (suitable for articles). We chose to hide the content rather than remove, or spend time trying to figure out better places in the docs, with the assumption that if it's important enough content, someone will circle back at some point to prioritize a better home.

## Using Fleet

### How can I switch to Fleet from Kolide Fleet?

To migrate to Fleet from Kolide Fleet, please follow the steps outlined in the [Upgrading Fleet section](https://fleetdm.com/docs/deploying/upgrading-fleet) of the documentation.

### Has anyone stress-tested Fleet? How many hosts can the Fleet server handle?

Fleet has been stress-tested to 150,000 online hosts and 400,000 total enrolled hosts. Production deployments exist with over 100,000 hosts, and numerous production deployments manage tens of thousands of hosts.

It’s standard deployment practice to have multiple Fleet servers behind a load balancer. However, typically, the MySQL database is the performance bottleneck, and a single Fleet server can handle tens of thousands of hosts.

### Can I target my hosts using their enroll secrets?

No, currently, there’s no way to retrieve the name of the enroll secret with a query. This means that there's no way to create a label using your hosts' enroll secrets and then use this label as a target for live queries or scheduled queries.

Typically, folks will use some other unique identifier to create labels that distinguish each type of device. As a workaround, [Fleet's manual labels](https://fleetdm.com/docs/using-fleet/fleetctl-cli#host-labels) provide a way to create groups of hosts without a query. These manual labels can then be used as targets for queries.

There is, however, a way to accomplish this even though the answer to the question remains "no": Teams. As of Fleet v4.0.0, you can group hosts in Teams either by enrolling them with a team-specific secret or by transferring hosts to a team. One the hosts you want to target are part of a team, you can create a query and target the team in question.

### How often do labels refresh? Is the refresh frequency configurable?

The update frequency for labels is configurable with the [—osquery_label_update_interval](https://fleetdm.com/docs/deploying/configuration#osquery-label-update-interval) flag (default 1 hour).

### Can I modify built-in labels?

While it is possible to modify built-in labels using `fleetctl` or the REST API, doing so is not recommended because it can lead to errors in the Fleet UI.
Find more information [here](https://github.com/fleetdm/fleet/issues/12479).

### How do I revoke the authorization tokens for a user?

Authorization tokens are revoked when the “require password reset” action is selected for that user. User-initiated password resets do not expire the existing tokens.

### How do I monitor the performance of my queries?

Fleet can live query the `osquery_schedule` table. Performing this live query allows you to get the performance data for your scheduled queries. Also, consider scheduling a query to the `osquery_schedule` table to get these logs into your logging pipeline.

### How do I monitor a Fleet server?

Fleet provides standard interfaces for monitoring and alerting. See the [Monitoring Fleet](https://fleetdm.com/docs/using-fleet/monitoring-fleet) documentation for details.

### Why is the “Add User” button disabled?

The “Add User” button is disabled if SMTP (email) has not been configured for the Fleet server. Currently, there is no way to add new users without email capabilities.

One way to hack around this is to use a simulated mail server like [Mailhog](https://github.com/mailhog/MailHog). You can retrieve the email that was “sent” in the Mailhog UI and provide users with the invite URL manually.

### Can I disable password-based authentication in the Fleet UI?

Some folks like to enforce users with SAML SSO enabled to log in only via the SSO and not via password.

There is no option in the Fleet UI for disabling password-based authentication.
However, users that have SSO enabled in Fleet will not be able to log in via password-based authentication.

If a user has SSO enabled, the Login page in the Fleet UI displays the “Email” and “Password” fields, but on attempted password-based login, this user will receive an “Authentication failed” message.

### Where are my query results?

#### Live queries

Live query results (executed in the web UI or `fleetctl query`) are pushed directly to the UI where the query is running. The results never go to a file unless you as the user manually save them.

#### Scheduled queries

Scheduled query results from enrolled hosts can be logged by Fleet.
For results to go to Fleet, the osquery `--logger_plugin` flag must be set to `tls`.

#### What are my options for storing the osquery logs?

Folks typically use Fleet to ship logs to data aggregation systems like Splunk, the ELK stack, and Graylog.

Fleet supports multiple logging destinations for scheduled query results and status logs. The `--osquery_result_log_plugin` and `--osquery_status_log_plugin` can be set to:
`filesystem`, `firehose`, `kinesis`, `lambda`, `pubsub`, `kafkarest`, and `stdout`.
See:
  - https://fleetdm.com/docs/deploying/configuration#osquery-result-log-plugin.
  - https://fleetdm.com/docs/deploying/configuration#osquery-status-log-plugin.

#### Troubleshooting

Expecting results but not seeing anything in the logs?

- Try scheduling a query that always returns results (eg. `SELECT * FROM time`).
- Check whether the query is scheduled in differential mode. If so, new results will only be logged when the result set changes.
- Ensure that the query is scheduled to run on the intended platforms and that the tables queried are supported by those platforms.
- Use live query to `SELECT * FROM osquery_schedule` to check whether the query has been scheduled on the host.
- Look at the status logs provided by osquery. In a standard configuration, these are available on the filesystem of the Fleet server at the path configurable by [`--filesystem_status_log_file`](https://fleetdm.com/docs/deploying/configuration#filesystem-status-log-file). This defaults to `/tmp/osquery_status`. The host will output a status log each time it executes the query.

### Why does the same query come back faster sometimes?

Don't worry; this behavior is expected. It's part of how osquery works.

Fleet and osquery work together by communicating with heartbeats. Depending on how close the next heartbeat is, Fleet might return results a few seconds faster or slower.
>To get around what's known as the "[thundering herd problem](https://en.wikipedia.org/wiki/Thundering_herd_problem#:~:text=In%20computer%20science%2C%20the%20thundering,but%20only%20one%20will%20win.)," these heartbeats aren't exactly the same number of seconds apart each time. Osquery implements a "splay", a few ± milliseconds that are added to or subtracted from the heartbeat interval to prevent these thundering herds. This helps prevent situations where many thousands of devices might unnecessarily attempt to communicate with the Fleet server at exactly the same time. (If you've ever used Socket.io, a similar phenomenon can occur with that tool's automatic WebSocket reconnects.)

### Why don't my query results appear sorted based on the ORDER BY clause I specified in my SQL query?

When a query executes in Fleet, the query is sent to all hosts at the same time, but results are returned from hosts at different times. In Fleet, results are shown as soon as Fleet receives a response from a host. Fleet does not sort the overall results across all hosts (the sort UI toggle is used for this). Instead, Fleet prioritizes speed when displaying the results.  This means that if you use an `ORDER BY` clause selection criteria in a query, the results may not initially appear with your desired order. However, the sort UI toggle allows you to sort by ascending or descending order for any of the displayed columns.

### What happens if I have a query on a team policy and I also have it scheduled to run separately?

Both queries will run as scheduled on applicable hosts. If there are any hosts that both the scheduled run and the policy apply to, they will be queried twice.

### Why aren’t my live queries being logged?

Live query results are never logged to the filesystem of the Fleet server. See [Where are my query results?](#where-are-my-query-results).

### Why does my query work locally with osquery but not in Fleet?

If you're seeing query results using `osqueryi` but not through Fleet, the most likely culprit is a permissions issue. Check out the [osquery docs](https://osquery.readthedocs.io/en/stable/deployment/process-auditing/#full-disk-access) for more details and instructions for setting up Full Disk Access.

### Can I use the Fleet API to fetch results from a scheduled query?

You cannot. Scheduled query results are logged to whatever logging plugin you have configured and are not stored in the Fleet DB.

However, the Fleet API exposes a significant amount of host information via the [`api/v1/fleet/hosts`](https://fleetdm.com/docs/using-fleet/rest-api#list-hosts) and the [`api/v1/fleet/hosts/{id}`](https://fleetdm.com/docs/using-fleet/rest-api#get-host) API endpoints. The `api/v1/fleet/hosts` [can even be configured to return additional host information](https://github.com/fleetdm/fleet/blob/9fb9da31f5462fa7dda4819a114bbdbc0252c347/docs/1-Using-Fleet/2-fleetctl-CLI.md#fleet-configuration-options).

For example, let's say you want to retrieve a host's OS version, installed software, and kernel version:

Each host’s OS version is available using the `api/v1/fleet/hosts` API endpoint. [Check out the API documentation for this endpoint](https://fleetdm.com/docs/using-fleet/rest-api#list-hosts).

It’s possible in Fleet to retrieve each host’s kernel version, using the Fleet API, through `additional_queries`. The Fleet configuration options YAML file includes an `additional_queries` property that allows you to append custom query results to the host details returned by the `api/v1/fleet/hosts` endpoint. For more information on setting up `additional_queries`, see the [configuration documentation](https://fleetdm.com/docs/configuration/configuration-files#features-additional-queries).

### How does Fleet set the `name` attribute in query automations?

Query automations use the format `pack/Global/<Your query name>` to set the `name` attribute, where `<Your query name>` is the exact name you assign to your query. 

### Why is my host not updating a policy's response?

The following are reasons why a host may not be updating a policy's response:

* The policy's query includes tables that are not compatible with this host's platform. For example, if your policy's query contains the [`apps` table](https://osquery.io/schema/5.0.1/#apps), which is only compatible with hosts running macOS, this policy will not update its response if this host is running Windows or Linux.

* The policy's query includes invalid SQL syntax. If your policy's query includes invalid syntax, this policy will not update its response. You can check the syntax of your query by heading to the **Queries** page, selecting your query, and then selecting "Save."

### What should I do if my computer is showing up as an offline host?

If your device is showing up as an offline host in the Fleet instance, and you're sure that the computer has osquery running, we recommend trying the following:

* Try unenrolling and re-enrolling the host. You can do this by uninstalling osquery on the host and then enrolling your device again using one of the [recommended methods](https://fleetdm.com/docs/using-fleet/adding-hosts).

### How does Fleet deal with IP duplication?

Fleet relies on UUIDs, so any overlap with host IP addresses should not cause a problem. The only time this might be an issue is if you are running a query that involves a specific IP address that exists in multiple locations, as it might return multiple results - [Fleet's teams feature](https://fleetdm.com/docs/using-fleet/teams) can be used to restrict queries to specific hosts.

### Can fleetd run alongside osquery?

Yes, fleetd can be run alongside an existing, separately-installed osqueryd. If you have an existing osqueryd installed on a given host, you don't have to remove it prior to installing fleetd.  The osquery instance provided by fleetd uses its own database directory that doesn't interfere with other osquery instances installed on the host.

### Can I control how fleetd handles updates?

Yes, auto-updates can be disabled entirely by passing `--disable-updates` as a flag when running `fleetctl package` to generate your installer (easy) or by deploying a modified systemd file to your hosts (more complicated). We'd recommend the flag:

```sh
fleetctl package --fleetctl package --type=deb --fleet-url=https://localhost:8080 --enroll-secret=superRandomSecret --disable-updates
```

You can also indicate the [channels you would like Fleetd to watch for updates](https://fleetdm.com/docs/using-fleet/fleetd#update-channels) using the `--orbit-channel`, `--desktop-channel`, and `--osqueryd-channel` flags:

```sh
fleetctl package --fleetctl package --type=deb --fleet-url=https://localhost:8080 --enroll-secret=superRandomSecret --orbit-channel=edge --desktop-channel=stable --osqueryd-channel=4
```

You can specify a major (4), minor (4.0), or patch (4.6.0) version, as well as the `stable`  or `edge` channels.

## When will the newest version of osquery be available to Fleetd?

When a new osquery version is released, it is pushed to the `edge` channel for beta testing. As soon as that version is deemed stable by the osquery project, it is moved to the `stable` channel. Some versions may take a little longer than others to be tested and moved from `edge` to `stable`, especially when there are major changes.

## Where does Fleetd get update information?

Fleetd checks for update metadata and downloads binaries at `tuf.fleetctl.com`.

## Can I bundle osquery extensions into Fleetd?

This isn't supported yet, but we're working on it!

### What happens to osquery logs if my Fleet server or my logging destination is offline?

If Fleet can't send logs to the destination, it will return an error to osquery. This causes osquery to retry sending the logs. The logs will then be stored in osquery's internal buffer until they are sent successfully, or they expire if the `buffered_log_max`(defaults to 1,000,000 logs) is exceeded. Check out the [Remote logging buffering section](https://osquery.readthedocs.io/en/latest/deployment/remote/#remote-logging-buffering) on the osquery docs for more on this behavior.

### How does Fleet work with osquery extensions?

Any extension table available in a host enrolled to Fleet can be queried by Fleet. Note that the "compatible with" message may show an error because it won't know your extension table, but the query will still work, and Fleet will gracefully ignore errors from any incompatible hosts.

### Why do I see an "Unknown Certificate Error" when adding hosts to my dev server?

If you are using a self-signed certificate on `localhost`, add the  `--insecure` flag when building your installation packages:

```sh
fleetctl package --fleetctl package --type=deb --fleet-url=https://localhost:8080 --enroll-secret=superRandomSecret --insecure
```

### Can I hide known vulnerabilities that I feel are insignificant?

This isn't currently supported, but we're working on it! You can track that issue [here](https://github.com/fleetdm/fleet/issues/3152).

### Can I create reports based on historical data in Fleet?

Currently, Fleet only stores the current state of your hosts (when they last communicated with Fleet). The best way at the moment to maintain historical data would be to use the [REST API](https://fleetdm.com/docs/using-fleet/rest-api) or the [`fleetctl` CLI](https://fleetdm.com/docs/using-fleet/fleetctl-cli) to retrieve it manually. Then save the data you need to your schedule.

### When do I need fleetctl vs. the REST API vs. the Fleet UI?

[fleetctl](https://fleetdm.com/docs/using-fleet/fleetctl-cli) is great for users who like to do things in a terminal (like iTerm on a Mac). Lots of tech folks are real power users of the terminal. It is also helpful for automating things like deployments.

The [REST API](https://fleetdm.com/docs/using-fleet/rest-api) is somewhat similar to fleetctl, but it tends to be used more by other computer programs rather than human users (although humans can use it too). For example, our [Fleet UI](https://fleetdm.com/docs/using-fleet/rest-api) talks to the server via the REST API. Folks can also use the REST API if they want to build their own programs that talk to the Fleet server.

The [Fleet UI](https://fleetdm.com/docs/using-fleet/fleet-ui) is built for human users to make interfacing with the Fleet server user-friendly and visually appealing. It also makes things simpler and more accessible to a broader range of users.


### Why can't I run queries with `fleetctl` using a new API-only user?

In versions prior to Fleet 4.13, a password reset is needed before a new API-only user can perform queries. You can find detailed instructions for setting that up [here](https://github.com/fleetdm/fleet/blob/a1eba3d5b945cb3339004dd1181526c137dc901c/docs/Using-Fleet/fleetctl-CLI.md#reset-the-password).

### Can I audit actions taken in Fleet?

The [REST API `activities` endpoint](https://fleetdm.com/docs/using-fleet/rest-api#activities) provides a full breakdown of actions taken on queries, policies, and teams (Available in Fleet Premium) through the UI, the REST API, or `fleetctl`.

### How often is the software inventory updated?

By default, Fleet will query hosts for software inventory hourly. If you'd like to set a different interval, you can update the [periodicity](https://fleetdm.com/docs/deploying/configuration#periodicity) in your vulnerabilities configuration.

### Can I group results from multiple hosts?

There are a few ways you can go about getting counts of hosts that meet specific criteria using the REST API. You can use [`GET /api/v1/fleet/hosts`](https://fleetdm.com/docs/using-fleet/rest-api#list-hosts) or the [`fleetctl` CLI](https://fleetdm.com/docs/using-fleet/fleetctl-cli#available-commands) to gather a list of all hosts and then work with that data however you'd like. For example, you could retrieve all hosts using `fleetctl get hosts` and then use `jq` to pull out the data you need. The following example would give you a count of hosts by their OS version:

```sh
$ fleetctl get hosts --json | jq '.spec .os_version' | sort | uniq -c

   1 "CentOS Stream 8.0.0"
   2 "Ubuntu 20.4.0"
   1 "macOS 11.5.2"
   1 "macOS 11.6.3"
   1 "macOS 12.1.0"
   3 "macOS 12.2.1"
   3 "macOS 12.3.0"
   6 "macOS 12.3.1"
```

### How do I downgrade from Fleet Premium to Fleet Free?

> If you'd like to renew your Fleet Premium license key, please contact us [here](https://fleetdm.com/company/contact).

**Back up your users and update all team-level users to global users**

1. Run the `fleetctl get user_roles > user_roles.yml` command. Save the `user_roles.yml` file so that, if you choose to upgrade later, you can restore user roles.
2. Head to the **Settings > Users** page in the Fleet UI.
3. For each user that has any team listed under the **Teams** column, select **Actions > Edit**, then select **Global user**, and then select **Save**. If a user shouldn't have global access, delete this user.

**Move all team-level scheduled queries to the global level**

1. Head to the **Schedule** page in the Fleet UI.
2. For each scheduled query that belongs to a team, copy the name in the **Query** column, select **All teams** in the top dropdown, select **Schedule a query**, past the name in the **Select query** field, choose the frequency, and select **Schedule**.
3. Delete each scheduled query that belongs to a team because they will no longer run on any hosts following the downgrade process.

**Move all team-level policies to the global level**

1. Head to the **Policies** page in the Fleet UI.
2. For each policy that belongs to a team, copy the **Name**, **Description**, **Resolve**, and **Query**. Then, select **All teams** in the top dropdown, select **Add a policy**, select **Create your own policy**, paste each item in the appropriate field, and select **Save**.
3. Delete each policy that belongs to a team because they will no longer run on any hosts following the downgrade process.

**Back up your teams**

1. Run the `fleetctl get teams > teams.yml` command. Save the `teams.yml` file so that, if you choose to upgrade later, you can restore teams.
2. Head to the **Settings > Teams** page in the Fleet UI.
3. Delete all teams. This will move all hosts to the global level.

**Remove your Fleet Premium license key**

1. Remove your license key from your Fleet configuration. Documentation on where the license key is located in your configuration is [here](https://fleetdm.com/docs/deploying/configuration#license).
2. Restart your Fleet server.

### If I use a software orchestration tool (Ansible, Chef, Puppet, etc.) to manage agent options, do I have to apply the same options in the Fleet UI?

No. The agent options set using your software orchestration tool will override the default agent options that appear in the **Settings > Organization settings > Agent options** page. On this page, if you hit the **Save** button, the options that appear in the Fleet UI will override the agent options set using your software orchestration.

### How does Fleet determine online and offline status?

#### Online hosts

**Online** hosts will respond to a live query.

A host is online if it has connected successfully in a window of time set by `distributed_interval` (or `config_tls_refresh`, whichever is smaller).
A buffer of 60 seconds is added to the calculation to avoid unnecessary flapping between online/offline status (in case hosts take a bit longer than expected to connect to Fleet).
The values for `distributed_interval` and `config_tls_refresh` can be found in the **Settings > Organization settings > Agent options** page for global hosts
and in the **Settings > Teams > TEAM NAME > Agent options** page for hosts that belong to a team.

For example:

`distributed_interval=10, config_tls_refresh=30`
A host is considered online if it has connected to Fleet in the last 70 (10+60) seconds.

`distributed_interval=30, config_tls_refresh=20`
A host is considered online if it has connected to Fleet in the last 80 (20+60) seconds.

#### Offline hosts

**Offline** hosts won't respond to a live query. These hosts may be shut down, asleep, or not connected to the internet.
A host could also be offline if there is a connection issue between the osquery agent running in the host and Fleet (see [What should I do if my computer is showing up as an offline host?](#what-should-i-do-if-my-computer-is-showing-up-as-an-offline-host)).

### Why aren't "additional queries" being applied to hosts enrolled in a team?

Changes were introduced in Fleet v4.20.0 that caused the `features.additional_queries` set in at the global level to no longer apply to hosts assigned to a team. If you would like those queries to be applied to hosts assigned to a team, you will need to include these queries under `features.additional_queries` in each team's [configuration](https://fleetdm.com/docs/using-fleet/configuration-files#teams).

### Why am I seeing an error when using the `after` key in `api/v1/fleet/hosts`?

There is a [bug](https://github.com/fleetdm/fleet/issues/8443) in MySQL validation in some versions of Fleet when using the `created_at` and `updated_at` columns as `order_key` along with an `after` filter. Adding `h.` to the column in `order_key` will return your results.

```text
{host}/api/v1/fleet/hosts?order_key=h.created_at&order_direction=desc&after=2022-10-22T20:22:03Z

```
### What can I do if Fleet is slow or unresponsive after enabling a feature?

Depending on your infrastructure capabilities and the number of hosts enrolled in your Fleet instance, Fleet might be slow or unresponsive after globally enabling a feature like [software inventory](https://fleetdm.com/docs/deploying/configuration#software-inventory).

In those cases, we recommend a slow rollout by partially enabling the feature by teams using the `features` key of the [teams configuration](https://fleetdm.com/docs/using-fleet/configuration-files#teams).

### Why am I getting errors when generating a .msi package on my M1 Mac?

There are many challenges to generating .msi packages on any OS but Windows. Errors will frequently resolve after multiple attempts, and we've added retries by default in recent versions of `fleetctl package`.  Package creation is much more reliable on Intel Macs, Linux, and Windows.

### Where did the "Packs" page go?

Packs are a function of osquery that provide a portable format to import/export queries in and out of platforms like Fleet. The "Packs" section of the UI that began with `kolide/fleet` c. 2017 was an early attempt at fulfilling this vision, but it soon became clear that it wasn't the right interface for segmenting and targeting hosts in Fleet.

Instead, 2017 "packs" functionality has been combined with the concept of queries. Queries now have built-in schedule features and (in Fleet Premium) can target specific groups of hosts via teams.

The "Packs" section of the UI has been removed, but access via the API and CLI is still available for backward compatibility. The `fleetctl upgrade-packs` command can be used to convert existing 2017 "packs" to queries.

Read more about osquery packs and Fleet's commitment to supporting them [here](https://fleetdm.com/handbook/company/why-this-way#why-does-fleet-support-query-packs).


### What happens when I turn off MDM?

In the Fleet UI, you can turn off MDM for a host by selecting **Actions > Turn off MDM** on the **Host details** page.

When you turn off MDM for a host, Fleet removes the enforcement of all macOS settings for that host. Also, the host will stop receiving macOS update reminders via Nudge. Turning MDM off doesn't remove the fleetd agent from the host. To remove the fleetd agent, share [these guided instructions](#how-can-i-uninstall-fleetd) with the end user.

To enforce macOS settings and send macOS update reminders, the host has to turn MDM back on. Turning MDM back on for a host requires end user action.

### What does "package root files: heat failed" mean?
We've found this error when you try to build an MSI on Docker 4.17. The underlying issue has been fixed in Docker 4.18, so we recommend upgrading. More information [here](https://github.com/fleetdm/fleet/issues/10700)


## Deployment

- [How do I get support for working with Fleet?](#how-do-i-get-support-for-working-with-fleet)
- [Can multiple instances of the Fleet server be run behind a load-balancer?](#can-multiple-instances-of-the-fleet-server-be-run-behind-a-load-balancer)
- [Why aren't my osquery agents connecting to Fleet?](#why-arent-my-osquery-agents-connecting-to-fleet)
- [How do I fix "certificate verify failed" errors from osqueryd?](#how-do-i-fix-certificate-verify-failed-errors-from-osqueryd)
- [What do I need to do to change the Fleet server TLS certificate?](#what-do-i-need-to-do-to-change-the-fleet-server-tls-certificate)
- [How do I migrate hosts from one Fleet server to another (eg. testing to production)?](#how-do-i-migrate-hosts-from-one-fleet-server-to-another-eg-testing-to-production)
- [What do I do about "too many open files" errors?](#what-do-i-do-about-too-many-open-files-errors)
- [Can I skip versions when updating Fleet to the latest version?](#can-i-skip-versions-when-updating-to-the-latest-version)
- [I upgraded my database, but Fleet is still running slowly. What could be going on?](#i-upgraded-my-database-but-fleet-is-still-running-slowly-what-could-be-going-on)
- [Why am I receiving a database connection error when attempting to "prepare" the database?](#why-am-i-receiving-a-database-connection-error-when-attempting-to-prepare-the-database)
- [Is Fleet available as a SaaS product?](#is-fleet-available-as-a-saas-product)
- [What MySQL versions are supported?](#what-mysql-versions-are-supported)
- [What are the MySQL user access requirements?](#what-are-the-mysql-user-requirements)
- [Does Fleet support MySQL replication?](#does-fleet-support-mysql-replication)
- [What is duplicate enrollment and how do I fix it?](#what-is-duplicate-enrollment-and-how-do-i-fix-it)
- [What API endpoints should I expose to the public internet?](#what-api-endpoints-should-i-expose-to-the-public-internet)
- [What Redis versions are supported?](#what-redis-versions-are-supported)
- [Will my older version of Fleet work with Redis 6?](#will-my-older-version-of-fleet-work-with-redis-6)

### How do I get support for working with Fleet?

For bug reports, please use the [Github issue tracker](https://github.com/fleetdm/fleet/issues).

For questions and discussion, please join us in the #fleet channel of [osquery Slack](https://fleetdm.com/slack).

### Can multiple instances of the Fleet server be run behind a load balancer?

Yes. Fleet scales horizontally out of the box as long as all of the Fleet servers are connected to the same MySQL and Redis instances.

Note that osquery logs will be distributed across the Fleet servers.

Read the [performance documentation](https://fleetdm.com/docs/using-fleet/monitoring-fleet#fleet-server-performance) for more.

### Why aren't my osquery agents connecting to Fleet?

This can be caused by a variety of problems. The best way to debug is usually to add `--verbose --tls_dump` to the arguments provided to `osqueryd` and look at the logs for the server communication.

#### Common problems

- `Connection refused`: The server is not running or is not listening on the address specified. Is the server listening on an address that is available from the host running osquery? Do you have a load balancer that might be blocking connections? Try testing with `curl`.
- `No node key returned`: Typically, this indicates that the osquery client sent an incorrect enroll secret that was rejected by the server. Check what osquery is sending by looking in the logs near this error.
- `Certificate verify failed`: See [How do I fix "certificate verify failed" errors from osqueryd](#how-do-i-fix-certificate-verify-failed-errors-from-osqueryd).
- `Bad record MAC`: When generating your certificate for your Fleet server, ensure you set the hostname to the FQDN or the IP of the server. This error is common when setting up Fleet servers and accepting defaults when generating certificates using `openssl`.

### How do I fix "certificate verify failed" errors from osqueryd?

Osquery requires that all communication between the agent and Fleet is over a secure TLS connection. For the safety of osquery deployments, there is no (convenient) way to circumvent this check.

- Try specifying the path to the full certificate chain used by the server using the `--tls_server_certs` flag in `osqueryd`. This is often unnecessary when using a certificate signed by an authority trusted by the system but is mandatory when working with self-signed certificates. In all cases, it can be a useful debugging step.
- Ensure that the CNAME or one of the Subject Alternate Names (SANs) on the certificate matches the address at which the server is being accessed. If osquery connects via `https://localhost:443`, but the certificate is for `https://fleet.example.com`, the verification will fail.
- Is Fleet behind a load-balancer? Ensure that if the load-balancer is terminating TLS, this is the certificate provided to osquery.
- Does the certificate verify with `curl`? Try `curl -v -X POST https://fleetserver:port/api/v1/osquery/enroll`.

### What do I need to do to change the Fleet server TLS certificate?

If both the existing and new certificates verify with osquery's default root certificates (such as a certificate issued by a well-known Certificate Authority) and no certificate chain was deployed with osquery, there is no need to deploy a new certificate chain.

If osquery has been deployed with the full certificate chain (using `--tls_server_certs`), deploying a new certificate chain is necessary to allow for verification of the new certificate.

Deploying a certificate chain cannot be done centrally from Fleet.

### How do I use a proxy server with Fleet?

Seeing your proxy's requests fail with an error like `DEPTH_ZERO_SELF_SIGNED_CERT`)?
To get your proxy server's HTTP client to work with a local Fleet when using a self-signed cert, disable SSL / self-signed verification in the client.

The exact solution to this depends on the request client you are using. For example, when using Node.js ± Sails.js, you can work around this in the requests you're sending with `await sails.helpers.http.get()` by lifting your app with the `NODE_TLS_REJECT_UNAUTHORIZED` environment variable set to `0`:

```sh
NODE_TLS_REJECT_UNAUTHORIZED=0 sails console
```

### I'm only getting partial results from live queries

Redis has an internal buffer limit for pubsub that Fleet uses to communicate query results. If this buffer is filled, extra data is dropped. To fix this, we recommend disabling the buffer size limit. Most installs of Redis should have plenty of spare memory to not run into issues. More info about this limit can be found [here](https://redis.io/topics/clients#:~:text=Pub%2FSub%20clients%20have%20a,64%20megabyte%20per%2060%20second.) and [here](https://raw.githubusercontent.com/redis/redis/unstable/redis.conf) (search for client-output-buffer-limit).

We recommend a config like the following:

```
client-output-buffer-limit pubsub 0 0 60
```

### How do I migrate hosts from one Fleet server to another (e.g. testing to production)?

Primarily, this would be done by changing the `--tls_hostname` and enroll secret to the values for the new server. In some circumstances (see [What do I need to do to change the Fleet server TLS certificate?](#what-do-i-need-to-do-to-change-the-fleet-server-tls-certificate)), it may be necessary to deploy a new certificate chain configured with `--tls_server_certs`.

These configurations cannot be managed centrally from Fleet.

### What do I do about "too many open files" errors?

This error usually indicates that the Fleet server has run out of file descriptors. Fix this by increasing the `ulimit` on the Fleet process. See the `LimitNOFILE` setting in the [example systemd unit file](https://fleetdm.com/docs/deploying/configuration#runing-with-systemd) for an example of how to do this with systemd.

Some deployments may benefit by setting the [`--server_keepalive`](https://fleetdm.com/docs/deploying/configuration#server-keepalive) flag to false.

This was also seen as a symptom of a different issue: if you're deploying on AWS on T-type instances, there are different scenarios where the activity can increase and the instances will burst. If they run out of credits, then they'll stop processing, leaving the file descriptors open.

### Can I skip versions when updating Fleet to the latest version?

Absolutely! If you're updating from the current major release of Fleet (v4), you can install the [latest version](https://github.com/fleetdm/fleet/releases/latest) without upgrading to each minor version along the way. Just make sure to back up your database in case anything odd does pop up!

If you're updating from an older version (we'll use Fleet v3 as an example), it's best to take some stops along the way:

1. Back up your database.
2. Upgrade to the last release of v3 - [3.13.0](https://github.com/fleetdm/fleet/releases/tag/3.13.0).
3. Migrate the database.
4. Test
5. Check the release post for [v4.0.0 ](https://github.com/fleetdm/fleet/releases/tag/v4.0.0) to see the breaking changes and get Fleet ready for v4.
6. Upgrade to v4.0.0.
7. Migrate the database.
8. Test
9. Upgrade to the [current release](https://github.com/fleetdm/fleet/releases/latest).
10. One last migration.
11. Test again for good measure.

Taking it a bit slower on major releases gives you an opportunity to better track down where any issues may have been introduced.

### I upgraded my database, but Fleet is still running slowly. What could be going on?

This could be caused by a mismatched connection limit between the Fleet server and the MySQL server that prevents Fleet from fully utilizing the database. First [determine how many open connections your MySQL server supports](https://dev.mysql.com/doc/refman/8.0/en/too-many-connections.html). Now set the [`--mysql_max_open_conns`](https://fleetdm.com/docs/deploying/configuration#mysql-max-open-conns) and [`--mysql_max_idle_conns`](https://fleetdm.com/docs/deploying/configuration#mysql-max-idle-conns) flags appropriately.

### Why am I receiving a database connection error when attempting to "prepare" the database?

First, check if you have a version of MySQL installed that is at least 8.0. Then, make sure that you currently have a MySQL server running.

The next step is to make sure the credentials for the database match what is expected. Test your ability to connect to the database with `mysql -u<username> -h<hostname_or_ip> -P<port> -D<database_name> -p`.

If you're successfully connecting to the database and still receive a database connection error, you may need to specify your database credentials when running `fleet prepare db`. It's encouraged to put your database credentials in environment variables or a config file.

```sh
fleet prepare db \
    --mysql_address=<database_address> \
    --mysql_database=<database_name> \
    --mysql_username=<username> \
    --mysql_password=<database_password>
```

### Is Fleet available as a SaaS product?

Yes! Please sign up for the [Fleet Cloud Beta](https://kqphpqst851.typeform.com/to/yoo5smT9).

### What MySQL versions are supported?

Fleet is tested with MySQL 8.0.36 and 8.4.2. Newer versions of MySQL 8 typically work well. AWS Aurora requires at least version 3.07.0. Please avoid using MariaDB or other MySQL variants that are not officially supported. Compatibility issues have been identified with MySQL variants, and these may not be addressed in future Fleet releases.

### What are the MySQL user requirements?

The user `fleet prepare db` (via environment variable `FLEET_MYSQL_USERNAME` or command line flag `--mysql_username=<username>`) uses to interact with the database needs to be able to create, alter, and drop tables as well as the ability to create temporary tables.

### Does Fleet support MySQL replication?

You can deploy MySQL or Maria any way you want. We recommend using managed/hosted mysql so you don't have to think about it, but you can think about it more if you want. Read replicas are supported. You can read more about MySQL configuration [here](https://fleetdm.com/docs/deploying/configuration#mysql).

### What is duplicate enrollment, and how do I fix it?

Duplicate host enrollment is when more than one host enrolls in Fleet using the same identifier
(hardware UUID or osquery generated UUID).

Typically, this is caused by cloning a VM Image with an already enrolled
osquery client, which results in duplicate osquery generated UUIDs. To resolve this issue, it is
advised to configure `--osquery_host_identifier=uuid` (which will use the hardware UUID) and then
delete the associated host in the Fleet UI.

In rare instances, VM Hypervisors have been seen to duplicate hardware UUIDs. When this happens,
using `--osquery_host_identifier=uuid` will not resolve the duplicate enrollment problem. Sometimes, the problem can be resolved by setting `--osquery_host_identifier=instance` (which will use the
osquery-generated UUID) and then deleting the associated host in the Fleet UI.

Find more information about [host identifiers here](https://fleetdm.com/docs/deploying/configuration#osquery-host-identifier).

### How do I resolve an "unknown column" error when upgrading Fleet?

The `unknown column` error typically occurs when the database migrations haven't been run during the upgrade process.

Check out the [documentation on running database migrations](https://fleetdm.com/docs/deploying/upgrading-fleet#running-database-migrations) to resolve this issue.

### What API endpoints should I expose to the public internet?

If you would like to manage hosts that can travel outside your VPN or intranet, we recommend only exposing the osquery endpoints to the public internet:

- `/api/osquery`
- `/api/v1/osquery`

If you are using Fleet Desktop and want it to work on remote devices, the bare minimum API to expose is `/api/latest/fleet/device/*/desktop`. This minimal endpoint will only provide the number of failing policies.

For full Fleet Desktop and scripts functionality, `/api/fleet/orbit/*` and`/api/fleet/device/ping` must also be exposed.

If you would like to use the fleetctl CLI from outside of your network, the following endpoints will also need to be exposed for `fleetctl`:

- `/api/setup`
- `/api/v1/setup`
- `/api/latest/fleet/*`
- `/api/v1/fleet/*`

If you would like to use Fleet's MDM features, the following endpoints need to be exposed:

- `/mdm/apple/scep` to allow hosts to obtain a SCEP certificate.
- `/mdm/apple/mdm` to allow hosts to reach the server using the MDM protocol.
- `/api/mdm/apple/enroll` to allow DEP-enrolled devices to get an enrollment profile.
- `/api/*/fleet/device/*/mdm/apple/manual_enrollment_profile` to allow manually enrolled devices to
  download an enrollment profile.

> The `/mdm/apple/scep` and `/mdm/apple/mdm` endpoints are outside of the `/api` path because they
> are not RESTful and are not intended for use by API clients or browsers.

### What is the minimum version of MySQL required by Fleet?

Fleet requires at least MySQL version 8.0.36, and is tested [with versions 8.0.36 and 8.4.2](https://github.com/fleetdm/fleet/blob/main/.github/workflows/test-go.yaml#L47)

### How do I migrate from Fleet Free to Fleet Premium?

To migrate from Fleet Free to Fleet Premium, once you get a Fleet license, set it as a parameter to `fleet serve` either as an environment variable using `FLEET_LICENSE_KEY` or in the Fleet's config file. See [here](https://fleetdm.com/docs/deploying/configuration#license) for more details. Note: You don't need to redeploy Fleet after the migration.

### What Redis versions are supported?
Fleet is tested with Redis 5.0.14 and 6.2.7. Any version of Redis after version 5 will typically work well.

### Will my older version of Fleet work with Redis 6?

Most likely, yes! While we'd definitely recommend keeping Fleet up to date in order to take advantage of new features and bug patches, most legacy versions should work with Redis 6. Just keep in mind that we likely haven't tested your particular combination, so you may run into some unforeseen hiccups.

## What happened to the "Schedule" page?
Scheduled queries are not gone! Instead, the concept of a scheduled query has been merged with a saved query. After 4.35, scheduling now happens on the queries page: a query can be scheduled (via familiar attributes such as "interval" and "platform"), or it can simply be saved to be run ad-hoc. A query can now belong to a team, or it can be a global query that every team inherits. This greatly simplifies the mental model of the product and enables us to build [exciting features](https://github.com/fleetdm/fleet/issues/7766) on top of the new unified query concept.

To achieve the above, 4.35 implemented an automatic migration which transitions any pre-existing scheduled query and [2017 pack](https://fleetdm.com/handbook/company/why-this-way#why-does-fleet-support-query-packs) into the new merged query concept:
- Any global scheduled query will have its query converted into a global query with the relevant schedule attributes (frequency, min. osquery version, logging, etc.).
- Any team-specific scheduled query will be converted into a query on that team with the relevant schedule characteristics.
- Any query that is referenced by a 2017 pack will be converted into a global query, and the 2017 pack will reference it. The 2017 packs should continue functioning as before.

Important: To avoid naming conflicts, a query must have a unique name within its team. Therefore, the migration will add a timestamp after each migrated query. If you are using gitops for queries, we recommend that you run `fleetctl get queries --yaml` after the migration to get the latest set of yaml files. Otherwise, if you run `fleetctl apply -f queries.yml`, it will result in the creation of new queries rather than updating the existing ones. To prevent this issue, we recommend you use `PATCH /api/v1/fleet/queries/{id}` for updating or changing query names.

For any automated workflows that use the schedule endpoints on the API, we recommend consolidating to the query endpoints, which now accept the scheduled query attributes. The schedule endpoints in the API still function but are deprecated. To accommodate the new unified query concept, the schedule endpoints behave differently under the hood:
- The POST endpoints will create a new query with the specified attributes
- The PATCH endpoint will modify the specified query with the specified attributes
- The DELETE endpoint will delete the specified query.

Finally, "shard" has been retired as an option for queries. In its place, we recommend using a canary team or a live query to test the impact of a query before deploying it more broadly.
-->


<meta name="description" value="Commonly asked questions and answers about deployment from the Fleet community.">

# Debugging

## Goals of this guide

This is NOT meant to be an exhaustive list of all possible issues in Fleet and how to solve them.

This is a guide for going from a vague statement such as "things are not working correctly" to a more narrowed down and 
specific assessment. This doesn't necessarily mean a solution, but with a more specific assessment, it'll be easier for 
the Engineering team to help.

Note that even if you follow all those steps, the Engineering team might have follow-up questions.

## Basic data that is needed

While it's not needed strictly 100% of the time, in most cases, it's extremely useful to have a clear understanding of 
the basic characteristics of the Fleet deployment with the issues:

- Amount of total hosts.
- Amount of online hosts.
- Amount of scheduled queries.
- Amount and size (CPU/Mem) of the Fleet instances.
- Fleet instances CPU and Memory usage while the issue has been happening.
- MySQL flavor/version in use.
- MySQL server capacity (CPU/Mem).
- MySQL CPU and Memory usage while the issue has been happening.
- Are MySQL read replicas configured? If so, how many?
- Redis version and server capacity (CPU/Mem).
- Is Redis running in cluster mode?
- Redis CPU and Memory usage while the issue has been happening.
- The output of `fleetctl debug archive`.

## Triaging the issue

The first step in understanding an issue better is figuring out in what area of the system the issue is happening. There 
are two main areas an issue might fall in: server-side or client-side.

A server-side issue is one where one of the few pieces of infrastructure on the server encounters an issue. Some of 
these pieces are: the MySQL database, the load balancer, a Fleet server instance.

A client-side issue is one where the issue occurs on the software that runs on the hosts (i.e. the machine that runs 
osquery/orbit/Fleet Desktop).

There are issues that expand both areas, but in most cases, the issue happens in one area, and the other is more of a 
symptom rather than the issue itself. So we'll continue this text with the assumption that multi-area issues are rare, 
and even if facing them, the following should help narrow it down.  

While classifying client and server-side issues is easy, it's also not realistic. So let's expand the 
categories a bit more and let's "mark" them with keyword:

1. Fleet itself (the binary/docker image running the Fleet API): `SERVER`
   1. A specific part of the Fleet UI is slow: `PARTIALSERVER`
2. MySQL: `MYSQL`
3. Redis: `REDIS`
4. Infrastructure: `INFRA`
5. osquery / orbit / Fleet Desktop: `OSQUERY`

With these areas in mind, here's a list of possible issues and what you should look into:

- A specific device (or a handful of devices) is not behaving as expected -> `OSQUERY`
- A specific device appears online, but the last fetch at is old -> `OSQUERY`
- The Fleet UI is slow overall -> `SERVER`
- A specific page (or a handful of pages, but not all) in the Fleet UI is slow -> `PARTIALSERVER`
- New devices cannot enroll -> `OSQUERY`
- Live query results come in very slowly -> `REDIS` or `SERVER`
- osquery Extensions are not working correctly -> `OSQUERY`
- fleetctl is getting errors when applying YAMLs -> `SERVER` 
- Migrations are taking too long -> `MYSQL`
- I see connection/network errors on the fleetctl or osquery logs, but not on my Fleet logs -> `INFRA`

### SERVER

Whenever diagnosing a server-side issue, one of the first steps is to look at Fleet itself. In particular, that means 
looking at the logs across all running intances. How to look at these logs would vary depending on your 
deployment. If, for example, it's an AWS deployment, and you're using our terraform files as guidance, you'd use 
CloudWatch.

Fleet, by default, will log errors, which are the first thing for which to look. If you have debug logging enabled, you can 
filter errors by filtering the keyword `err`.

These logs will be the first way to triage a server-side error. For example, if there are timeouts happening in APIs, 
you should continue by looking at `MYSQL` and then `REDIS`. Otherwise, if it seems like a more illustrative error, this 
would be a good point to reach out with all the information gathered.

If there are no errors in the logs and everything looks normal, check `INFRA`.

### PARTIALSERVER

Sometimes Fleet operates without any errors but accessing a specific part of the web UI is slow. As a starting point it
would be good to get a screenshot of the Network tab in your browser’s Developer Tools. The primary data that needs to 
be visible are Name, Status, and Time (in Chrome's terms). 
[Here's how to accomplish this using Google Chrome](https://developer.chrome.com/docs/devtools/network/).

Besides this, it might be good to continue with `MYSQL` and `REDIS`.

Depending on the API, there will likely be follow-up questions about the amount of data, but this would be a good point to 
check in with Engineering.

### MYSQL

Most of the data needed to understand an issue in MySQL should've been gathered already by the basic data specified at 
the beginning of this document. However, there is a chance that Fleet is running with a database user that is not 
capable of querying the information needed. So here are the queries that would output a good first step in information
gathering:

```sql
show engine innodb status;
show processlist;
```

If read replicas are configured, another piece of important data is whether there has been any replication lag 
registered.

With all this gathered, it's a good time to reach out to the Engineering team.

### REDIS

In most cases, the data gathered at the beginning of this document should be enough to understand what might be 
happening with Redis. However, if more information is needed, running the 
[monitor command](https://redis.io/commands/monitor/) should shed more light on the issue.

**WARNING**: if Redis is suffering from performance issues, running monitor will only increase the problem.

A less invasive way to check for more stats, if Elasticache is being used (or another system with more reporting), other 
metrics like current connections, replication lag if applicable, if one instance is largely overused compared to others 
in cluster mode, the number of commands per key type could help identify what is wrong.

### OSQUERY

Just like with the Fleet server, the best way to understand issues on the client-side is to look at logs.

If you are running vanilla osquery in the host, please restart the host with `--tls_dump` and `--verbose`. This will 
allow us to see more details as to what's happening in the communication with Fleet (or lack thereof). Check the 
[official documentation](https://osquery.readthedocs.io/en/stable/deployment/logging/) for details about locating 
the logs and other configurations.

If you are running Fleetd, you should add `--debug` to the command-line options. This will get debug logs for Fleetd and 
also for osquery automatically. Check the [fleetd docs](https://fleetdm.com/docs/using-fleet/enroll-hosts#finding-fleetd-logs) 
for more details as to where to find Fleetd-specific logs.

If you're having issues with memory leaks or failures in the orbit process, then you can create a profile dump.
This includes information about memory allocations and thread creation in order to debug excessive memory allocation.
They should not include any sensitive data, such as tokens. You can see more information at the [official go page](https://go.dev/doc/diagnostics#profiling).
In order to create dumps, run `pkill -USR1 orbit` as root on macOS or Linux. The profiles should show up in a ZIP
file in `/opt/orbit/profiles`, named according to the time they were created.

If you are running Fleet Desktop, there's no change needed, you should see the log file in the following directories 
depending on the platform:

- Linux: `$XDG_STATE_HOME/Fleet` or `$HOME/.local/state/Fleet`
- macOS: `$HOME/Library/Logs/Fleet`
- Windows: `%LocalAppData%/Fleet`

The log file name is `fleet-desktop.log`.

If the issue is related to osquery extensions, the following data would be needed:

- osquery version
- OS it's running on
- What does the extension do?
- How is the extension queried/deployed?
- In what language is the extension implemented?
- What's the nature of the problem? (i.e. whether the extension is respawning, the extension can’t connect, 
or the extension is up/working and then dies and can’t reconnect)

With this data, it's time to reach out to Engineering.

### `INFRA`

At this level, you want to look into Load Balancer logs, errors, and configurations. For instance, does the LB 
have a request size limit? If the LB is not terminating TLS, is that appropriately configured on the Fleet side?

Make sure as well that your cloud provider is not having issues of their own. For instances, check
[AWS](https://health.aws.amazon.com/health/status) for status.


<meta name="maintainedBy" value="lukeheath">
<meta name="description"  value="A guide to triaging and diagnosing issues in Fleet.">

# Rethinking endpoint management: Fleet, osquery and Infrastructure as Code

Traditional MDM and EDR tools demand blind trust. It’s time to adopt an engineering approach to device management that prioritizes visibility, auditability, and scale

### Links to article series:

- Part 1: [The confidence gap: why IT leaders are abandoning legacy endpoint management](https://fleetdm.com/articles/the-confidence-gap)
- Part 2: Rethinking endpoint management: Fleet, osquery and Infrastructure as Code
- Part 3: [Supercharging endpoint management: AI assistants and event-driven automation](https://fleetdm.com/articles/supercharging-endpoint-management)

## The black box

For decades, IT and security leaders have entered into an uncomfortable bargain with endpoint management vendors. You buy a proprietary "black box" (a Mobile Device Management (MDM) solution or an Endpoint Detection and Response (EDR) agent), install it on your thousands of devices, and hope it does what the sales brochure promised.

You trust that it’s patching correctly. You trust that it’s detecting the right threats. You trust that the vendor’s private API isn’t doing something it shouldn't.

But in an era of zero-day vulnerabilities, distributed workforces, and heterogeneous OS environments, **blind trust is no longer a viable strategy**. Modern organizations need to move beyond "Click-Ops" (manual, error-prone processes for managing devices via a web GUI) and adopt an engineering-driven approach. 

Why embrace automation? Because it’s relatively easy to reverse version-controlled automation, and very hard to unclick buttons.

Implementing this idea requires a fundamental shift in the stack. It means moving away from opaque, proprietary tools and toward a transparent, composable architecture built on three pillars: 

- **Comprehensive endpoint visibility**  
- **API-driven automation**  
- **Infrastructure as code (IAC) workflows**

Here is why forward-thinking IT leaders are rebuilding their endpoint strategy around this open foundation.

## The foundation of truth

If you cannot see it, you cannot manage it. 

Traditional client-side agents provide a filtered, pre-determined view of device data based on what the vendor thinks is important. Furthermore, the data is often hours or even days old. To manage your configuration, you must **know** what your complete configuration actually is in real time. This is a solved problem.

`osquery` changes the paradigm. It is an open-source instrumentation framework that expresses operating system data (Windows, macOS, Linux, ChromeOS) as a high-performance relational database. It allows you to ask questions about your devices using `SQLite` syntax.

Instead of running complex, brittle PowerShell or bash scripts to check the status of a firewall or look for a rogue process, you use a query:

`SELECT * FROM firewall_rules;`

or

`SELECT name, pid FROM processes WHERE name = 'suspicious_binary';`

**The Value for IT Leaders:** `osquery` provides universal, kernel-level visibility that is vendor-agnostic. It turns endpoint telemetry into structured data that your team already knows how to get and, if they don’t, Fleet makes it easy to upload pre-built queries from trusted [sources](https://hub.com/palantir/osquery-configuration/tree/master/Fleet/Endpoints) based on security standards like [CIS](https://fleetdm.com/guides/cis-benchmarks#basic-article) & the NIST Security Compliance Project. It eliminates the reliance on proprietary vendor dashboards for basic truths about your infrastructure.

## The orchestration layer

`osquery` is powerful, but it is a single-host tool. You cannot manually `ssh` into 50,000 laptops to run `SQL` queries. You need a control plane to manage deployments, schedule queries, collect results, and take action. Most importantly, the way you manage the control plane must enable trusted and validated automation of all actions. In order to interact and manage a fleet of devices, the solution must make it easy to automate all changes, which means it needs to expose all the APIs so that automated actions are first-class citizens as much as clicking around in the UI.

This is [**Fleet**](https://fleetdm.com/).

Fleet is the most widely used open-source control plane for `osquery`. It is designed to scale from a startup's first ten Macs to an enterprise's 300,000 mixed-OS servers and workstations. Fleet is designed and architected "API-first" - all of Fleet’s features and functions are available 1:1 in the GUI and the API.

- **Fleet Free** provides the essential tools for querying devices in real-time and piping that data into your SIEM or data lake (like Snowflake, Splunk, or ELK) for analysis.  
- **Fleet Premium adds MDM** upgrades the stack from observation to fully-operational multi-platform device management for Apple, Linux, Windows, Android & ChromeOS. [Fleet premium](https://fleetdm.com/pricing) allows organizations to enforce disk encryption, push software profiles, manage updates, and remotely wipe devices - all using the same lightweight, robust agent that provides the best device visibility solution available.

**The Value for IT Leaders:** Fleet consolidates your tool sprawl. Instead of separate agents for observation, inventory data collection, software vulnerability detection, MDM and compliance enforcement, you have one platform. Because Fleet’s core is open-source, you are never locked into a black box. The API is open, the code is auditable, and the roadmap is transparent.

## The management philosophy: Infrastructure as Code (IaC)

This is where the true revolution happens. Managing `osquery` with Fleet is powerful, but if your admins are still logging into a web console to manually toggle settings for thousands of devices, you have only incrementally improved a broken process.

To achieve true scale and reliability, endpoint management must be treated like software engineering. It must be managed using **Infrastructure as Code (IaC)** principles. In an IaC workflow with Fleet, you don't define a security policy by clicking buttons in a GUI. You define it in a simple, human-readable text file in [`YAML`](https://yaml.org/) syntax and stored in a `git` repository.

Let's look at an example:

A policy requiring FileVault encryption on macOS can be controlled with a checkbox in the Fleet GUI. 

But, that checkbox can also be declaratively controlled with a text file containing the FileVault configuration. 

By adding the text file into a version-controlled  `git` repository, the "code" is "merged" into the "main" code branch - the "source of truth" that stores all of the latest repository updates. The merge action kicks off a [CI / CD pipeline](https://about.gitlab.com/topics/ci-cd/cicd-pipeline/) automation using Fleet’s API via [`fleetctl`](https://fleetdm.com/guides/fleetctl#basic-article) (Fleet's GitOps-native CLI binary for controlling the Fleet UI). This action updates Fleet and pushes configurations out to managed devices in scope.

**The Value for IT Leaders:** IaC introduces engineering rigor to IT operations.

1. **Auditability & Compliance:** Every change to your device configurations is tracked in `git`. You know exactly who changed a firewall policy, when they did it, and why (via the commit message). Compliance audits go from weeks of archaeology to hours of reviewing `git` history.  
2. **Peer Review & Safety:** No lone admin can accidentally push a bad configuration that bricks half your fleet on a Friday afternoon. Changes require pull requests and peer reviews by senior engineers before they touch production devices.  
3. **Disaster Recovery & Repeatability:** If your management environment went down today, could you rebuild it exactly as it was? With IaC, your entire configuration is backed up in `git`. You can spin up a new environment and re-apply your state in minutes.

## The shift to engineering-driven IT

The combination of the ground truth from `osquery`, Fleet’s scalable orchestration, and the rigor of Infrastructure as Code provides something traditional vendors cannot: complete control over your environment. Fleet transforms device management from manual labor in the GUI into proactive infrastructure engineering with all the benefits that GitOps entails: your teams can see every change, undo any error, and repeat every success. 

But, there is a catch: moving to code-based management requires new skills. How do you enable a standard IT admin to write `SQL` queries and `YAML` configurations without needing the expertise of a seasoned DevOps engineer?

In part 3 of this series, we will explore how emerging AI coding assistants and event-driven automation bridge that gap, making this powerful stack accessible to teams of any size.

<meta name="articleTitle" value="Rethinking endpoint management: Fleet, osquery and Infrastructure as Code">  
<meta name="authorFullName" value="Ashish Kuthiala, CMO, Fleet Device Management">  
<meta name="authorGitHubUsername" value="akuthiala">  
<meta name="category" value="articles">  
<meta name="publishedOn" value="2026-02-26">  
<meta name="description" value="Part 2 of 3 - Article series on supercharging modern endpoint management.">

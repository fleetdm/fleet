# Development groups

Fleet organizes development groups by their goals. These include members from Design, Engineering, and Product.

Goals:

_progress (+) guarantee_

- **Interface** - more, successfully adopted features faster
  - (+) keep UI & API simple, minimalist, consistent, and bug-free
- **Platform** - improve the productivity of the Interface team through patterns and infrastructure for implementing new features, reduce REST API latency, increase max load test size, make upgrading seamless for users, improve accuracy and reliability of data
  - (+) maintain quick time-til-merge timeframe for PRs reviewed, and maintain clean, empathetic interfaces that allow contributors in other groups to execute quickly and without the need to wait for review or approvals
- **Agent**: grow # open source, osquery-based agents by making Fleet’s agents better, faster, and broader in capabilities
  - (+) every table works intuitively with user-friendly docs and empathetic caveats, warnings, and error messages

At Fleet, groups define the relevant sections of the engineering org chart.  A product manager (PM) represents each group and reports to the Product department (or a founder serving as an
interim product manager):

- Interface PM: Noah Talerman
- Platform PM: Mo Zhu
- Agent PM: Mo Zhu

Each group's PM works closely with engineers within their group:
- The PM **prioritizes** work and defines **what** to iteratively build and release next within their group's domain to best serve the group's goals and the company's goals as a whole. The PM communicates **why** this work is prioritized and works with engineering to come up with the best possible **how**.
- The PM is responsible for the execution of initiatives (epics) that span across multiple groups. These epics are tracked as GitHub issues with the "epic" label. One PM is assigned to the epic to make sure that all issues associated with the epic (child issues) make it into a release.

An engineering manager (EM), with their group of engineers, forms the engineering members of the group: 

- Interface EM: Luke Heath
- Platform EM: Tomás Touceda
- Agent EM: Zach Wasserman

Each group's EM works closely with the PM and engineers in their group:
- The EM (along with engineers) defines **how** to implement that definition within the surface area of code, processes, and rituals owned by their group while serving their group’s goals and the company's goals as a whole.
- The EM is responsible for the execution of epics within their group. One EM is assigned to the epic to make sure that all child issues make it into a release.

## Interface group
### Responsibilities
- Everything related to Fleet's graphical user interface (other than for the desktop application portion of Fleet Desktop)
- `fleetctl` (the Fleet command-line interface) and the associated YAML documents (almost everything in fleetctl besides the `fleetctl debug` subcommands)
- The REST API that serves these
- The UX/developer experience, flow, steps, and associated UI and API interfaces for how integrations that require any user interaction or configuration (e.g., GeoIP, Zendesk, Jira), including which third-party integrations are supported and which API styles and versions are chosen
- End to end testing of the application (e.g., Cypress)
- The REST API documentation
- Future officially-supported wrapper SDKs, such as the Postman collection or, e.g., a Python SDK
- Fleet's configuration surface, including
  - The config settings that exist for Fleet deployments and how they're configured
  - How feature flags are used
  - Their default values, supported data types, and error messages
  - Associated documentation on fleetdm.com
  - The UX of upgrading and downgrading and sidegrading Fleet tiers, and managing license keys

### Consumers
- A human using Fleet's graphical user interface
- A human who is writing code that integrates Fleet's REST API 
- A human reading Fleet's REST API docs
- A human using fleetctl, Fleet's Postman collection or Fleet's other future SDK wrappers

These humans might be working within the "Interface" group itself insofar as they consume the Fleet REST API.  Or they might be a contributor to the Fleet community.  Or one of Fleet's core users or customers, usually in an SRE, IT, or security role in an organization.

### Goals
- Bring value to Fleet users by delivering new features and iterations of existing features.
- Increase adoption and stickiness of features.
- Keep the graphical user interface, REST API, fleetctl, and SDKs like Postman reliable, minimal, consistent, and easy to use.
- Promote stability of the API, introducing breaking changes only through the documented [API versioning](https://fleetdm.com/docs/contributing/api-versioning#what-kind-of-versioning-will-we-use-for-the-api) strategy or at major version releases.
- Ensure observance of semantic versioning for the Fleet API and config between releases so that only major versions include breaking changes.
- Delight users of Fleet's API, UI, SDKs, and documentation with a simple, secure, widely-adopted user and developer experience.
- **Improve Fleet’s feature value and ease of use.**

## Platform group
### Responsibilities
- The implementation of Fleet Agent API: i.e.,
  - The API used by agents on enrolled hosts to communicate with Fleet
  - The API used by agents for doing auto-updates and future installation/upgrade of custom extensions (e.g., TUF)
- Everything related to providing a stable, simple-to-build-on platform for Fleet contributors to use when implementing changes to the REST API
  - APIs for storing, retrieving, and modifying device data
  - APIs for running asynchronous and scheduled tasks
  - APIs for communicating with external services (e.g., HTTP, SMTP)
- The challenges of scale
  - Sometimes taking over development/improvement of features from the “Interface” group when these features have unexpected backend complexity or scaling challenges.
- Production infrastructure, including
  - Fleet Cloud demo
  - Fleet Cloud prod
  - The registry (TUF) used for auto-updates and (in the future) extensions
  - whatever backend is needed to generate installers in self-managed and hosted Fleet deployments
  - The future monitoring and 24/7/365 enhancement to the on-call rotation necessary for Fleet Cloud
- Behind-the-scenes integrations
  - i.e., integrations that make Fleet "just work" and don't involve configuration from users
  - Example: the code that fetches and manages CVE data from NVD and other behind-the-scenes infrastructure that enables vulnerability management to exist in Fleet without requiring any interactions or configuration from users
- The CI/CD pipeline
- `loadtest.fleetdm.com`
- `dogfood.fleetdm.com`

### Consumers
- A contributor from inside or outside the company.
- A host enrolled in Fleet running an osquery-based agent.
- The person who deploys, upgrades, and operates Fleet.
- A person who uses Fleet and expects it to be fast, reliable, and joyful.

### Goals
- Reduce REST API latency
- Increase max load test size
- Reduce infrastructure costs for Fleet deployments.
- Make upgrading Fleet versions seamless for users.
- Improve the integrity of data (both collected directly from agents or derived from that data, e.g., vulnerabilities).
- Maintain quick time-til-merge timeframe for PRs reviewed.
- Maintain clean, empathetic interfaces that allow contributors in other groups (or from outside the company) to execute quickly and without the need to wait for review or approval.
- **Make Fleet as easy as possible to operate and contribute to**

## Agent group
### Responsibilities
- osquery core, including the plugin interfaces and its config surface.
- Orbit
- Fleet Desktop
- Any future agent-based software built by Fleet.
- Fleet Agent API: the API interface and contributor docs used by osquery-enrolled agents for communicating with Fleet  (how to implement the internals is up to the "Platform" group).
- The extensions/tables are bundled in Orbit/Fleet Desktop, such as mac admins.

### Consumers
- An engineer working on a custom solution (usually built in-house) on top of osquery
- An SRE, IT operations, or DevOps professional using osquery-based agents in their default AMIs or container images, which deploys and manages osquery-based agents on their laptops, production servers/containers, and other corporate infrastructure
- An end-user running an osquery-based agent (Fleet Desktop, orbit, or osquery) on their work laptop, who wants their laptop to be stable, performant, and as private as possible
- An enterprise app owner (software engineer) running osquery-based agents on her app's servers
- A contributor working on vanilla osquery in osquery/osquery
- A contributor working on Fleet Desktop or orbit in fleetdm/fleet
- Fleet itself, consuming the data generated by osquery and any other agent software

### Goals
- Grow mind/market share of open source, osquery-based agents by making Fleet’s agents better, faster, and broader in capabilities
- Every table works intuitively with user-friendly docs and empathetic caveats, warnings, and error messages
  - Every table is documented within the Fleet UI and in fleetdm.com/docs, with GOTCHAS, deprecation notices, and the version when the table was added
- **Make Fleet’s agents easy to operate and contribute to**

<meta name="maintainedBy" value="mikermcneil">
<meta name="title" value="Development groups">

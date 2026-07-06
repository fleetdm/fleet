# The hidden cost of config-as-code: simplicity, tribal knowledge, and what stays in Git

*Config-as-code has won the argument. The one still worth having is about cost — in time, in skill, and in how much of your operation walks out the door when one person does.*

Config-as-code has quietly become the default expectation for device management. Jamf, Zentral, Workspace ONE, and Fleet all let you describe your endpoints in a repository, review changes through pull requests, and keep an audit trail of who changed what and why. That convergence is a good thing, and the argument about *whether* to manage devices as code is mostly over.

The argument still worth having is about cost — not the license cost, but the cost in time, in skill, and in dependence on a single person. Two config-as-code setups can deliver the same governance benefits and still ask very different things of the team that has to live with them. This isn't a knock on any tool. It's the question every IT leader should ask before committing: when the person who set this up moves on, does the knowledge leave with them, or does it stay in Git?

## Key takeaways

- **The debate that matters isn't code versus clicks.** Every serious platform now supports config-as-code, so the real decision is how much time, skill, and single-person dependence a given setup asks of you for the same governance benefits.
- **Terraform brings its whole world along, every day.** Reaching config-as-code through a provider means owning providers, state files, plan/apply, and reference graphs — real infrastructure work that stays whether or not the task in front of you needed it.
- **The simplest task reveals the gap.** Defining a group and scoping an app to it is a few readable lines of YAML in Fleet; the provider equivalent adds resources, IDs, and a reference graph you have to understand before the diff makes sense.
- **Simplicity widens who can participate.** GitOps only delivers if many people can safely read, approve, and ship a change — a workflow only one specialist can operate has just moved the old bottleneck into a state file.
- **You don't have to go all-in on day one.** Fleet's GitOps exceptions let you manage the parts you're confident about in Git while keeping others editable in the console, so the team can ramp instead of committing to a big-bang cutover.
- **Turnover is the real test.** When your expert leaves, a documented schema and a readable repository keep your institutional memory in Git; a workflow that depends on deep tooling expertise keeps it only as long as the expert stays.

<a purpose="cta-button" href="https://fleetdm.com/infrastructure-as-code">See config-as-code in Fleet</a>

## Two roads to the same place

There are broadly two ways device platforms approach config-as-code today.

The first builds the workflow into the product. You write configuration files in a documented schema, run a single command, and the platform reconciles itself to match. Fleet works this way. A set of YAML files describes the desired state of the instance, and `fleetctl gitops` (typically a one-line step in a GitHub Actions or GitLab pipeline) applies them. There's no extra tool to install and no separate state to manage. The repository is the source of truth, and the Fleet instance is the live state.

The second exposes the product through [Terraform](https://developer.hashicorp.com/terraform) or the drop-in replacement [Open Tofu](https://opentofu.org). You describe resources in HashiCorp Configuration Language (HCL), and a provider translates those resources into API calls. This is how config-as-code reaches Jamf Pro, Zentral, and, more recently, Workspace ONE. It's a legitimate and, in capable hands, powerful approach. Terraform is excellent software, and a mature provider like Zentral's slots cleanly into a team that already runs infrastructure-as-code.

Neither road is wrong. But they don't cost the same to travel, and the difference is easy to underestimate from a demo.

## What Terraform brings along

Using Terraform for device management means taking on the full set of concerns the tool was designed for, not just the device-level objects you're trying to change. That's not a criticism of the tool. It's simply what the tool and provider are built for. The conceptual surface includes:

- **Providers.** Each platform needs at least one provider to source, version-pin, authenticate, and keep current. In the Jamf world, full coverage can mean running two side by side: the community Jamf Pro provider for classic configuration and the Jamf Platform Terraform provider, published by Jamf Concepts under an open-source license, for modern features, bridged by a data source that translates one object's ID into another's. The Workspace ONE UEM provider, by contrast, is still in tech preview and scoped to macOS management today.  
- **State.** Terraform maintains a state file that maps your HCL to real-world objects. That file has to be stored, secured, locked, and backed up, usually in a remote backend. Mishandled state operations can orphan or delete real resources, and the state may contain secrets you don't want exposed.
- **Plan and apply.** Every run computes a plan describing what will be created, changed, or destroyed. Reading that plan safely, especially the destroy lines, is a skill, not a given.  
- **Lifecycle and references.** Resources reference each other by ID. Groups are likely addressed by numeric IDs or UUIDs you have to go find, rather than by name. Brownfield adoption means importing existing objects into state before you can manage them.

None of this is exotic to a platform engineer. It’s real work that infrastructure teams have done for over a decade, and it keeps coming: provider upgrades, breaking changes, drift, state hygiene. The point isn’t that it’s hard. The point is that it's *there*, to stay, every day, whether or not the smaller task in front of you needed it.

## A concrete comparison: assigning a group

Consider the most ordinary task in device management: define a group of devices and scope something to it.

In Fleet, a label is a few lines of YAML, and targeting it is a list of names:

```yml
labels:
  - name: Engineering
    description: Hosts used by engineers
    label_membership_type: host_vitals
    criteria:
      vital: end_user_idp_department
      value: Engineering
```

```yml
software:
  packages:
    - path: ../platforms/macos/packages/figma.package.yml
      labels_include_any:
        - Engineering
```

There's no separate assignment object, no ID to resolve, and no reference graph to reason about. A reviewer reads the diff and understands it immediately: this group is defined by IdP department, and this app now goes to it.

The Terraform equivalent does the same job, but the structure is heavier. A smart group is its own resource, and the thing being scoped references that group by ID rather than by name. Assignments live in nested blocks, and groups are typically addressed by numeric IDs or UUIDs rather than the readable names you'd use in YAML.

The gap isn't really about line count. It's about how much you need to know before the diff makes sense.

## Simplicity is a feature, not a compromise

It's tempting to read "simpler" as "less capable." In config-as-code, the opposite is usually true. The true value of GitOps comes from how many people can confidently participate in it. A workflow only one or two specialists can safely operate has quietly recreated the bottleneck GitOps was supposed to remove. It's just moved the bottleneck from a GUI into a state file.

When you evaluate a config-as-code approach, the most useful question isn't "what's the ceiling of what this can express?" It's "who on my team can read a change, approve it, and ship it without help?" If the honest answer is "whoever owns Terraform," that's worth knowing before you commit, not after.

This matters most for teams without a dedicated platform-engineering function, which is most IT teams. For those teams the setup cost of the built-in approach is minimal: "install the CLI, generate a starter repo, add a pipeline step, and place assets in a clear folder structure." The setup cost of the Terraform approach is a project: choose and pin providers, stand up remote state with locking, design a module and variable layout, build import flows for what already exists, and train people to read plans without fear. Both get you to config-as-code. One gets you there this week, with the whole team able to follow along.

## You don't have to go all in on day one: GitOps exceptions

There's a quieter objection underneath all of this, and it's the one that stops a lot of teams before they start: config-as-code feels like an all-or-nothing commitment. The moment you point a GitOps workflow at your instance, the repository becomes the source of truth, and anything you change in the GUI gets reverted on the next run. For a team that wants the benefits of GitOps but isn't ready to move *everything* into Git yet, that's a real deterrent. Most adoption stories don't begin with the whole estate in version control. They begin with one or two things, and a lot of nervousness about the rest.

The honest answer is that you shouldn't have to choose between full GitOps and no GitOps. A good config-as-code workflow should let you phase it in: manage the parts you're confident about in Git, keep the rest editable in the console while you get comfortable, and move things over as the team is ready. The transition should meet you where you are, not demand a big-bang cutover.

This is exactly the friction Fleet's GitOps exceptions feature was built to remove, and it's worth noting it was built in response to customer feedback that all-or-nothing GitOps adoption was a barrier. Released in Fleet 4.84.0, GitOps exceptions let an admin opt specific resource types out of GitOps enforcement (software, labels, and enroll secrets), leaving those manageable through the UI or API while everything else stays governed by Git. The practical effect is a gradual ramp: start by managing policies and configuration profiles in Git, then fold in software and labels later once the team has its footing. Exceptions are configured per resource type and require global-admin permissions, and Fleet guards against the obvious foot-gun: if a resource is both excepted *and* defined in a YAML file, the GitOps dry run surfaces a clear error rather than silently overwriting your console-managed changes. ClickOps and GitOps coexist on purpose, for as long as you need them to.

That's the kind of detail that separates a workflow you can adopt from one you have to brace for. Lowering the bar to *start* matters as much as the ceiling of what the tool can eventually do, because the teams that benefit from GitOps are the ones who actually make the move, not the ones still waiting for a quarter clear enough to migrate everything at once.

## The real test: when people move on

Here's the scenario every IT leader should plan for, because it always happens. The person who set up your config-as-code pipeline leaves. The clever module structure, import quirks, why a resource is pinned to an old provider version, and the critical workaround that keeps the state file safe: where does that knowledge live?

If it lives in someone's head, you have tribal knowledge, and it walks out the door with them. If it lives in a documented schema and a readable repository the rest of the team already understands, it stays.

This is the quiet promise of config-as-code that's easy to lose sight of: when people move on, the knowledge stays in Git. Every change is recorded with its author, its timestamp, and (if you've used the workflow well) its reasoning in the pull request. A new hire can read the history and understand not just the current state but how it got there.

That promise is only as strong as the number of people who can read the repository. A simple, declarative schema keeps the promise. A workflow that depends on deep tooling expertise keeps it only as long as the expert stays. The simpler the surface, the more of your institutional memory actually survives turnover, which over a few years is most of the value.

## How to evaluate

You don't need a vendor to tell you which approach is right for you. A few honest questions will:

- **Who needs to participate?** Count the people who'll realistically read and approve changes. Then ask how many of them can do that today, and how many would need training first.  
- **What's the day-two cost?** Setup is a one-time expense. Maintenance is forever. Account for provider upgrades, state management, and drift, not just the first successful apply.  
- **Can you phase it in?** You shouldn't have to migrate everything before you get any value. Check whether the workflow lets you manage some things in Git while keeping others in the console, so the team can ramp at its own pace instead of committing to a big-bang cutover.  
- **What happens when your expert leaves?** Imagine your most knowledgeable person is gone next month. Can the rest of the team keep shipping changes safely? The answer tells you how much of your operation is really in Git versus in someone's memory.  
- **Does the complexity buy you something you need?** Sometimes the full power of Terraform is exactly right: multi-system orchestration, an existing IaC practice, a team fluent in it. If so, use it. The goal is a deliberate choice, not a default one.

## The bottom line

Config-as-code is the right direction for device management, no matter which road you take to get there. Both roads give you version history, peer review, and an audit trail. What differs is the overhead you carry along the way, and whether the people who come after you can pick up where you left off.

Tools should lower the barrier to participation, not raise it. The more your team can read, review, and ship changes without routing everything through one specialist, the more resilient your operation is, and the more of your hard-won knowledge stays put when someone moves on.

The best workflow is the one your whole team can still run after you're gone. Evaluate accordingly.

## See it live

The quickest way to see this in detail is to inspect our [it-and-security](https://github.com/fleetdm/fleet/tree/main/it-and-security) folder to review the config, then download the [fleetctl](https://fleetdm.com/download) CLI and run `fleetctl new` to generate the GitOps scaffold. If you’d like help getting started, two good next steps are:

- [**Get a demo**](https://fleetdm.com/contact)**.**  We'll walk through how config-as-code and GitOps exceptions would work in your environment.
- [**Join a GitOps training session**](https://fleetdm.com/gitops-workshop)**.** Learn to manage configuration as code: set up apps, configs, reports, and policies in Git, review changes with pull requests, and deploy through CI in our hands‑on workshop.

*Fleet is the open-source endpoint management platform for macOS, Windows, Linux, and more. Want to try GitOps in your own fleet?* [*Get a demo*](https://fleetdm.com/contact) *or explore the* [*GitOps reference*](https://fleetdm.com/docs/configuration/yaml-files)*.*

<meta name="articleTitle" value="The hidden cost of config-as-code: simplicity, tribal knowledge, and what stays in Git">
<meta name="authorFullName" value="Henry Stamerjohann">
<meta name="authorGitHubUsername" value="headmin">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-06-18">
<meta name="description" value="The most sustainable config-as-code strategy is one your entire team can manage, even without relying on a dedicated specialist.">

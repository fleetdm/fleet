# The confidence gap: why IT leaders are abandoning legacy endpoint management

Managing thousands of endpoints shouldn’t rely on blind trust or manual UI clicks. Here is why engineering-driven organizations are demanding a new standard for device management.

### Links to article series:

- Part 1: The confidence gap: why IT leaders are abandoning legacy endpoint management
- Part 2: [Rethinking endpoint management: Fleet, osquery and Infrastructure as Code](https://fleetdm.com/articles/rethinking-endpoint-management)
- Part 3: [Supercharging endpoint management: AI assistants and event-driven automation](https://fleetdm.com/articles/supercharging-endpoint-management)

## Assessing the damage

If you ask an IT or Security leader how many devices are in their fleet, they can usually give you an exact number. But if you ask them to prove that every one of those devices is encrypted, patched, and running the correct endpoint detection agent at this exact second, the answer usually involves a sigh and a spreadsheet.

This is the **confidence gap**.

It is the void between what your management tools *say* is happening and the actual ground truth of your infrastructure. For years, IT leaders have accepted this gap as a cost of doing business. They have relied on legacy Mobile Device Management (MDM) platforms that treat devices as black boxes and rely on opaque, proprietary reporting.

But as remote work scales and threat landscapes evolve, the margin for error has vanished. Engineering and IT teams are realizing that legacy MDM solutions are fundamentally misaligned with modern IT operations.

Here is a look at the three biggest challenges IT leaders face with legacy device management, how the confidence gap impacts operations, and how Fleet is solving these problems for some of the world’s top engineering teams.

## Challenge 1: The confidence gap and lack of ground truth

Traditional MDMs operate on a "fire and forget" model. You push a profile to enforce a firewall policy, the MDM reports "Success" and the dashboard turns green. But legacy tools often fail to detect configuration drift when device state changes leaving IT leaders with a false sense of security.

You cannot confidently manage what you cannot accurately observe. IT leaders need to query their devices like a database to get real-time, irrefutable proof of state.

**How Fleet Solves It:** Fleet uses `osquery` to provide absolute ground truth. Instead of relying on a vendor’s pre-packaged dashboard, IT teams can ask their fleet exact questions (e.g., "Show me all devices where the firewall is disabled *right now*") and get immediate, verifiable answers.

**From the field:** When [Fastly](https://www.fastly.com/) needed to secure both their corporate endpoints and their global CDN infrastructure, they realized their previous MDM couldn't provide the certainty their leadership required. By switching to Fleet, they achieved a unified security posture.

>"We have much better visibility of our endpoints with Fleet compared to our previous MDM... The shift to GitOps has modernized our operations giving us the agility and change control we needed, giving leadership real-time confidence in device health and compliance." **- Dan Jackson, Sr Manager Systems Engineering, [Fleet @ Fastly case study](https://fleetdm.com/case-study/fastly)**

## Challenge 2: The "Click-Ops" trap

Modern infrastructure teams manage servers, networks, and applications using Infrastructure as Code (IaC). Changes are saved as text files, versioned files are tracked in git, peer-reviewed, and deployed via CI/CD pipelines.

Yet, when it comes to managing employee devices, these same teams are forced backward into "Click-Ops." Legacy MDM requires administrators to manually log into web portals, click through drop-down menus, and toggle settings by hand to deploy profiles. This manual approach does not scale, lacks version control, and introduces potential for costly human error.

**How Fleet Solves It:** Fleet is built for an Infrastructure as Code (GitOps) workflow. IT teams can define their entire device management state - configuration profiles, software, policies, reports, scripts - in human-readable text files. Every change is observable, reversible, repeatable and easily integrated into existing CI/CD pipelines.

**From the field:** [Stripe](https://stripe.com/) views MDM as critical infrastructure. Managing 10,000 Macs manually was not an option for their engineering culture.

>"Stripe has a strong focus on security and automation. Their previous MDM relied on manual, UI-driven workflows, which didn't fit their 'infrastructure as code' approach. Fragmented APIs made it hard to automate even simple tasks... The MDM had to fit Stripe's automation workflows. Choosing Fleet gave Stripe additional confidence." **- [Fleet @ Stripe case study](https://fleetdm.com/case-study/stripe)**

## Challenge 3: Tool sprawl and OS silos

In a hybrid work environment, developers and employees need to use the operating systems that make them most productive—whether that is macOS, Windows, or Linux.

Legacy MDM vendors usually specialize in one ecosystem. This forces IT leaders to buy and support multiple tools to do the same job: an Apple-specific MDM for Macs, a separate tool for Windows, and often completely ignore their Linux workstations. The result is a fragmented IT environment with duplicated licensing costs, inconsistent security policies, and an IT team suffering from dashboard fatigue.

**How Fleet Solves It:** Fleet provides a single, unified control plane for macOS, Windows, Linux, iOS, Android, and ChromeOS. IT teams enroll devices into Fleet to gain visibility and manage configurations in one place, drastically reducing overhead and tool sprawl.

**From the field:** The small IT team at [Foursquare](https://foursquare.com/) was overwhelmed by maintaining multiple tools for their 200+ Macs and Windows devices.

>"As a small team, [we] were running two separate MDM platforms... The overhead of maintaining both systems was overwhelming and added unnecessary complexity. By switching to Fleet, Foursquare cut endpoint maintenance effort by 50% and achieved 114% ROI by removing duplicate tools." **- [Fleet @ Foursquare case study](https://fleetdm.com/case-study/foursquare)**

## Closing the gap

The confidence gap exists because legacy tools ask IT leaders to trust a black box. Engineering-driven organizations require more than trust. They require transparent, auditable, and scalable proof that their intended device posture is enabled.

To solve the challenges of limited visibility, error-prone manual actions, and tool sprawl, IT must treat endpoint management as an engineering discipline.

The [next article](https://fleetdm.com/articles/rethinking-endpoint-management) in this series will highlight the components that make up this modern endpoint management stack: `osquery`, [Fleet](https://fleetdm.com/), and [Infrastructure as Code (Iac)](https://about.gitlab.com/topics/gitops/infrastructure-as-code/). The [final article in the series](https://fleetdm.com/articles/supercharging-endpoint-management) will cover supercharging your IT team and launching endpoint management into the future by using AI coding assistants and event-driven automation.

<meta name="articleTitle" value="The confidence gap: why IT leaders are abandoning legacy endpoint management">  
<meta name="authorFullName" value="Ashish Kuthiala, CMO, Fleet Device Management">  
<meta name="authorGitHubUsername" value="akuthiala">  
<meta name="category" value="articles">  
<meta name="publishedOn" value="2026-02-25">  
<meta name="description" value="Part 1 of 3 - Article series on supercharging modern endpoint management.">

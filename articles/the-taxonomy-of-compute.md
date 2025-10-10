# The taxonomy of compute

I've been traveling a lot, and over the course of a few recent plane rides, I decided I'd try to dissect the device management solutions on the market versus how people are actually using them. Here's what I found.

There are many types of computing devices and other computing infrastructure out there in the world.

---

## What are all these devices for?

### Productivity endpoints

**Corporate-owned**
- Shared office Macs (e.g. mixing consoles)  
- Shared PCs (e.g. computer labs)  
- Employee-issued Macs  
- Employee-issued desktop Linux  
- Employee-issued Windows hardware  
- Contributor-issued Windows VDIs  
- Contributor-issued Chromebooks  
- Employee-issued business smartphones (iPhones, Android phones)  
- Contributor-issued field/rugged devices (iOS, Android)  

**Contributor-owned (BYOD)**
- Employee-owned desktop Linux  
- Employee-owned personal smartphones  
- Vendor-owned computers (e.g. hardware BPOs issue to their employees)  

---

### Infrastructure and other dedicated endpoints

**Kiosks and meeting room devices**
- Mac minis  
- iPads  
- Android tablets  
- Apple TV  
- Linux remote controls  

**In-store / in-warehouse devices**
- iOS  
- Android scanners  
- Point-of-sale (PoS) Android tablets  
- Point-of-sale (PoS) iPads  

**Production lines**
- Windows OT controllers  
- iPads and Android tablets  

**Cloud / datacenter infrastructure**
- Bare metal Linux servers  
- Linux virtual machines  
- Windows servers  
- Kubernetes containers  
- EC2 Mac  

---

## Platform variation

Further variation exists within individual computing platforms:

**Android**
- Commercial Android  
- AOSP/Zebra devices  

**Linux**
- Ubuntu  
- Arch Linux (e.g. Omarchy, Manjaro)  
- Linux Mint  
- openSUSE  
- Pop!_OS (e.g. System76)  
- CentOS  
- RHEL (Red Hat Enterprise Linux)  
- Fedora  
- Zorin OS  
- Debian  
- Kali Linux  

**Windows**
- Windows 11  
- Windows 8–10  
- Windows XP, Windows 7  
- Windows Server  

---

## It's simple, really!

All of these flavors of devices and infrastructure need to be kept secure, reliable, and productive. To do that, IT and security practices need to be mature and integrated across each of the 5 stages of the computing lifecycle:

1. **PROVISION**: Setup and enablement  
2. **CONTROL**: Security hardening, controls, & governance  
3. **AUDIT**: Compliance, reports, & monitoring  
4. **SUPPORT**: Patching, service, & maintenance  
5. **RECYCLE**: Device reassignment, inventory, & asset management  

---

## The state of solutions today

Many solutions exist to achieve some of this journey. Even the handful of complete solutions that span the entire computing lifecycle are specialized and limited to particular platforms. These existing solutions also tend to be closed, not open, and not always the most customer-centric or collaborative—often passed around in acquisition after acquisition.  

Many also have competing incentives, come with lock-in, or require complicated licensing that makes it harder to know what you're paying for and what you get.  

### Key players, by category

**Apple device management**
- Jamf (acq. Vista)  
- Workspace ONE (acq. KKR, fka. Airwatch)  
- Addigy  
- Miradore  

**Configuration management**
- Ansible  
- SaltStack  
- Chef  
- Puppet  

**Windows device management**
- Workspace ONE (acq. KKR, fka. Airwatch)  
- Microsoft SCCM/GPO/Intune  
- ManageEngine Mobile Device Manager Plus  
- JumpCloud  

**Windows patch management**
- Ivanti  
- PatchMyPC  
- BigFix  
- Automox  

**Vulnerability management**
- Rapid7 (InsightVM)  
- Qualys  
- Tenable (Nessus)  
- Crowdstrike Exposure Management (fka. Spotlight)  

---

## That's a lot of complexity.

So why haven't organizations consolidated, or at least simplified, their stacks?

### Why isn't this solved yet?

Many have tried. But historically there haven't been many good, modern options that are both enterprise-friendly and complete.  

To complicate matters further, these products tend to be bought in multi-year cycles, and replacing them requires change management and thoughtful migration—eating up valuable hours you could be spending doing other things.

> "Companies have 10–12 agents running on some systems, sapping CPU performance, running proprietary code, shipping sensitive logs to multiple vendors, complicating audits, interrupting the employee experience with notifications, and bloating the stack with overlapping functionality and spend. But all these agents are effectively doing the same thing we were doing in RadioShack BASIC back in 1995: PEEK and POKE."  
> *–Me, getting annoyed about all the stuff running on my computer*

---

## Fleet’s approach

At Fleet, we're building the first complete solution for every platform, for every stage of the entire computing lifecycle.

There's plenty of work still to do, but over the last several years, the community has added support for more and more of these platforms and use cases. Now, we're working on figuring out how best to present that growing maturity of the product, and how best to show where we still have work to do.

Let me know what you think!  
—Mike  

---

## Is it any good?

Fleet manages **over 2 million computing devices globally across 90 countries and 1,300+ customers**. It's designed to be transparent, outsider-friendly, and efficient for teams with advanced needs and large deployments.  

Internally, it's based on widely adopted, security-forward technology that gives you full control, leaving you free to support the choices that work for your organization.

> "I just moved 10,000 Macs to Fleet."  
> –Wes Whetstone, Client Platform Engineer at Stripe  

Unlike other solutions, Fleet works no matter where your computers live. You can have Fleet host it in the cloud for you or deploy it yourself, in any environment, without sacrificing features or support. (You get the same experience either way.)  

In fact, you can pick up and move your MDM server and security data anywhere, anytime, which comes in handy with **120+ countries now implementing data residency restrictions** and further geopolitical complexity brewing.

Fleet is also (as of today) the **first and only MDM server that explicitly prioritizes desktop Linux**, not just Apple, including support for:
- Remote lock/wipe  
- Escrowed disk encryption (LUKS)  
- Patch compliance  

Curious? You can read about how [Faire migrated 1,000 devices](#) or why hundreds of others from the community, like a top AI chip manufacturer, made the switch.

<meta name="articleTitle" value="The taxonomy of compute.">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="articles">
<meta name="publishedOn" value="2025-10-03">
<meta name="description" value="An overview of today’s complex device landscape and how Fleet unifies open, transparent management across every platform.">

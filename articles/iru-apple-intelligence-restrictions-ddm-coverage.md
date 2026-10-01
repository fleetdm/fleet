# Iru now blocks Siri AI and visual intelligence over DDM, and buyers will expect the same from every MDM

*Iru's September 30 release adds Apple Intelligence restrictions to its declarative device management coverage. Here's what it signals about where Apple management is heading, and how to prepare.*

## Key takeaways

- **A competitor just raised the baseline.** Iru, the rebranded Kandji, shipped restrictions on September 30 that let admins block specific Apple Intelligence features, including Siri AI and visual intelligence. Buyers comparing MDMs will start asking whether yours can do the same.
- **Apple Intelligence is becoming a policy category of its own.** Apple's enterprise release notes now list Siri AI and visual intelligence among the features device management can restrict, so "block it or allow it" is a per-feature decision instead of one switch.
- **Coverage follows what Apple exposes.** An MDM can only manage what Apple publishes, so the useful question is how quickly a vendor delivers each new Apple control and whether you can deliver it yourself in the meantime.
- **A setting you pushed isn't a setting that landed.** Iru's release also improves user-channel reporting, which points at the real gap: knowing which devices actually applied a restriction.
- **Fleet delivers Apple's declarations as code.** Custom declarations and profiles live in Git, go through review, and roll out by fleet, so a new Apple control doesn't have to wait for a product release.

<a purpose="cta-button" href="https://fleetdm.com/device-management">See how Fleet manages Apple devices</a>

On September 30, Iru announced new declarative device management (DDM) restrictions that let admins block specific Apple Intelligence features, such as Siri AI and visual intelligence. The release also expands Safari Extensions management and adds clearer user-channel reporting for MDM.

Taken alone, that is one vendor's release note. Taken with the rest of the market, it shows the direction. Apple keeps adding AI features, Apple keeps exposing management controls for them, and MDM vendors are racing to wire those controls up. This article looks at what that means for the way you plan Apple coverage.

## Apple Intelligence is now a per-feature decision

Early Apple Intelligence controls were coarse. Apple's current enterprise release notes are more specific, listing Siri AI, visual intelligence, and natural language calendar event editing as features device management can restrict. Iru's September 30 release builds on that list.

That changes the policy question. "Do we allow Apple Intelligence?" becomes "Which features are acceptable for which groups?" A legal team may want Siri AI off while an engineering team keeps it on. A kiosk fleet may block everything. Each answer needs a setting that can be scoped by group.

## Coverage depends on what Apple publishes

Every MDM works from the same source: the controls Apple documents. When Apple adds a restriction, vendors differ only in how quickly they surface it and how much room they leave you to act sooner.

When you compare MDMs on Apple Intelligence, ask two things. First, which of Apple's current controls does the product expose today? Second, if Apple ships a new one tomorrow, can you deploy it without waiting for the vendor? The second answer matters more over a year than the first does today.

## The reporting gap

Iru paired the restrictions with clearer user-channel MDM reporting. That pairing is telling. Pushing a restriction is the easy half. Confirming that every Mac, iPhone, and iPad applied it is where audits and security reviews tend to get stuck, especially for settings that apply per user instead of per device.

If you tell a reviewer that visual intelligence is blocked, you want to answer with data from the devices, not with the contents of a policy.

## Where Fleet fits

Fleet manages macOS, iOS, and iPadOS with declarative device management. You can upload a declaration or configuration profile in the Fleet UI, or keep it as a file in Git and deploy it with GitOps, then scope it to specific fleets or labels. If Apple publishes a new control, you can deliver it as a custom setting without waiting on a product release.

Fleet's agent can then report device state back, so you can check settings on the devices that should have them. Fleet doesn't have a dedicated Apple Intelligence screen, and the exact payload for each feature depends on what Apple documents for it, so check Apple's platform deployment documentation for the current keys.

## What to do now

Decide which Apple Intelligence features each group of devices should have, write those decisions down as code, and keep a way to verify them on the devices. Competitors will keep adding controls, and Apple will keep adding features. A process that treats both as routine will hold up better than one that waits for the next release note.

## See it live

- **Get a demo** to see declarations and verification across your own Apple devices: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read the primer on declarative device management:** [fleetdm.com/articles/declarative-device-management-a-primer](https://fleetdm.com/articles/declarative-device-management-a-primer)

<meta name="articleTitle" value="Iru now blocks Siri AI and visual intelligence over DDM, and buyers will expect the same from every MDM">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-01">
<meta name="description" value="Iru added Apple Intelligence restrictions to its DDM coverage on September 30. See what it signals for Apple management and how to verify them.">

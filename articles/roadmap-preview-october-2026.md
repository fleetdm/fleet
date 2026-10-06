# Roadmap preview, October 2026

<!-- TODO: Embed the walkthrough video after the HoIT review.
<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/VIDEO_ID" allowfullscreen></iframe>
</div>
-->

The Fleet roadmap is set for fall 2026. This quarter is about AI you can put to work and Windows you can fully manage. Watch the video above for a walkthrough, or continue reading for the highlights.

In the next 3 months, Fleet will ship...

- 🧠 Starting in October: Generate configuration profiles, including Windows CSPs, with Fleet's AI skill in Claude, Codex, or Copilot, or from Slack and Microsoft Teams. New GitOps repos get the skill out of the box with `fleetctl new` ([#53265](https://github.com/fleetdm/fleet/issues/53265))
- 🚀 Fleet 5 (December):
  - 🤖 AI-native configuration profile builder: Generate any kind of profile right in the Fleet UI. Describe what you want in plain English and get a validated profile for Apple, Windows, and Android. Use your company's approved LLM ([#51813](https://github.com/fleetdm/fleet/issues/51813), [#51979](https://github.com/fleetdm/fleet/issues/51979), [#54801](https://github.com/fleetdm/fleet/issues/54801), [#51815](https://github.com/fleetdm/fleet/issues/51815))
  - 👁️‍🗨️ AI governance: See AI tools, AI skills, and AI agent sessions on each host, and find vulnerable MCP servers ([#51599](https://github.com/fleetdm/fleet/issues/51599), [#51288](https://github.com/fleetdm/fleet/issues/51288), [#51383](https://github.com/fleetdm/fleet/issues/51383), [#51825](https://github.com/fleetdm/fleet/issues/51825), [#54548](https://github.com/fleetdm/fleet/issues/54548))
  - ✨ Fleet's MCP server, hosted for you in your Fleet instance ([#44448](https://github.com/fleetdm/fleet/issues/44448))
  - 🧹 A cleaner, more consistent API ([5.0.0 milestone](https://github.com/fleetdm/fleet/milestone/295))
- 🪟 Windows apps: Deploy Microsoft Store apps and `.zip` packages, and give end users Fleet Desktop on Windows ([#43493](https://github.com/fleetdm/fleet/issues/43493), [#38800](https://github.com/fleetdm/fleet/issues/38800), [#48755](https://github.com/fleetdm/fleet/issues/48755))
- 📋 Windows configuration profiles: Upload profiles exported from Intune, see the XML applied to each host, use IdP user variables, and resend profiles when variables change ([#48198](https://github.com/fleetdm/fleet/issues/48198), [#54471](https://github.com/fleetdm/fleet/issues/54471), [#50144](https://github.com/fleetdm/fleet/issues/50144), [#44852](https://github.com/fleetdm/fleet/issues/44852))
- 🖥️ Windows enrollment: Show a custom end user agreement (EULA) and match Autopilot hosts without relying on the serial number ([#50146](https://github.com/fleetdm/fleet/issues/50146), [#51180](https://github.com/fleetdm/fleet/issues/51180))
- 🛡️ Windows vulnerabilities: See OS, KB, and configuration findings on each host, plus .NET and Microsoft Defender CVEs ([#50096](https://github.com/fleetdm/fleet/issues/50096), [#53455](https://github.com/fleetdm/fleet/issues/53455), [#45633](https://github.com/fleetdm/fleet/issues/45633))
- 🔐 Conditional access: Okta on Windows and Linux, and require Fleet-managed hosts in Google ([#53284](https://github.com/fleetdm/fleet/issues/53284), [#53286](https://github.com/fleetdm/fleet/issues/53286), [#54888](https://github.com/fleetdm/fleet/issues/54888))
- 🔗 Asset management: Send host data to ServiceNow and Oomnitza ([#38864](https://github.com/fleetdm/fleet/issues/38864), [#38866](https://github.com/fleetdm/fleet/issues/38866))
- 📱 Android: Kiosk mode, app patching, OS updates, and zero-touch enrollment into the right fleet ([#51586](https://github.com/fleetdm/fleet/issues/51586), [#54356](https://github.com/fleetdm/fleet/issues/54356), [#54359](https://github.com/fleetdm/fleet/issues/54359), [#51479](https://github.com/fleetdm/fleet/issues/51479))
- 🩹 Patch policies: Set a deadline for macOS apps ([#39176](https://github.com/fleetdm/fleet/issues/39176))
- ⏰ macOS updates: Update to the latest version within a major version ([#45511](https://github.com/fleetdm/fleet/issues/45511))
- 🛍️ Self-service: Let macOS end users install opt-in configuration profiles ([#46834](https://github.com/fleetdm/fleet/issues/46834))
- 🍏 Enroll and manage tvOS ([#38791](https://github.com/fleetdm/fleet/issues/38791))
- 🧩 Host vitals: Pull any attribute from your IdP and create custom vitals ([#42922](https://github.com/fleetdm/fleet/issues/42922))
- 🏷️ Labels for mobile devices: Use built-in host vitals (e.g. public IP) to create labels for iOS/iPadOS and Android hosts ([#39088](https://github.com/fleetdm/fleet/issues/39088))
- 🗓️ Run scripts on a recurring schedule ([#29496](https://github.com/fleetdm/fleet/issues/29496))

Big opportunities that Fleet is building towards in the near future (next 180 days):

- 🪟 Windows enrollment: Authenticate end users with any IdP, no Entra license required ([#48338](https://github.com/fleetdm/fleet/issues/48338))
- ⏰ Patch deadlines and end user prompts for Windows apps ([#48756](https://github.com/fleetdm/fleet/issues/48756))
- 📦 Deploy large packages, like local LLMs, Microsoft Office, and Xcode ([#48900](https://github.com/fleetdm/fleet/issues/48900))
- 💬 Ask questions about your hosts in Slack with Fleet's Slack bot ([#50509](https://github.com/fleetdm/fleet/issues/50509))

Any feedback or questions? Contributions welcome! You can find us [where we hang out](https://fleetdm.com/support).

<meta name="category" value="announcements">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-10-13">
<meta name="articleTitle" value="Roadmap preview, October 2026">
<meta name="description" value="The product improvements Fleet is currently working on and the biggest open opportunities in the product in the near future.">

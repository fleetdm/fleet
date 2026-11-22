# Firefox Enterprise and the managed browser checklist for IT

*Mozilla's Firefox Enterprise console governs the browser. Installing, updating, and auditing Firefox on every computer is still the device team's job.*

## Key takeaways

- **Mozilla released Firefox Enterprise on October 12, 2026.** It pairs Firefox with a self-hosted console for policy, data loss prevention (DLP), AI controls, extensions, and security logs.
- **Mozilla says to keep your MDM.** Mobile device management (MDM) still installs, configures, and updates Firefox on each computer.
- **Firefox now ships a major release every two weeks.** A version check that passed last month may fail today.
- **Pricing, console OS support, and console licensing aren't public.** Plan your evaluation around those gaps.

<a purpose="cta-button" href="https://fleetdm.com/guides/how-to-use-policies-for-patch-management-in-fleet">See how patch policies keep Firefox current</a>

Mozilla released Firefox Enterprise on October 12, 2026, with sales starting in France, Germany, Austria, and Switzerland. The console controls what happens inside the browser. It doesn't install Firefox, update it, or prove what's on each computer. Before you evaluate it, confirm you can report the Firefox version, policies, and extensions on every device.

## What does Firefox Enterprise include?

[Mozilla's announcement](https://blog.mozilla.org/en/firefox/introducing-firefox-enterprise/) pairs Firefox with a management console you run on your own infrastructure. The [enterprise browser page](https://www.firefox.com/browsers/enterprise) lists:

- Centralized policy control.
- DLP rules that warn or block sensitive data actions by website.
- AI controls that allow, restrict, or disable AI tools.
- Export of browser events to a security information and event management (SIEM) system.
- Lockdown Mode for high-risk browsing.
- Organization-wide extension allow and block lists.

Mozilla frames the launch around digital sovereignty: control over where data and management systems live. Three details aren't public yet. Mozilla hasn't published pricing, the console's supported operating systems, or whether the console is open source ([It's FOSS](https://itsfoss.com/news/firefox-enterprise/)).

## Why does Firefox Enterprise still need device management?

Mozilla's announcement is direct: "Keep your identity provider. Keep your SIEM. Keep your MDM." The browser console and the device layer do different jobs.

| Task | Firefox Enterprise console | Fleet |
|---|---|---|
| Browser policy, DLP, AI controls, SIEM export | Yes | Delivers Firefox policies as macOS configuration profiles |
| Install Firefox or Firefox ESR | No | Fleet-maintained apps on macOS and Windows |
| Keep Firefox versions current | No | Patch policies on macOS and Windows |
| Report installed extensions | Sets allow and block rules | `firefox_addons` table on macOS, Windows, and Linux |

The sovereignty question applies to device data too. [Fleet for Europe](https://fleetdm.com/europe) runs hosted in the EU, in your own cloud, on-premises, or air-gapped. Fleet is ISO/IEC 27001 certified, and its agent and server are open source.

## The managed browser checklist

- [ ] **Choose a channel for each group.** The standard release ships every two weeks, starting with Firefox 155 in September 2026. Firefox ESR (Extended Support Release) ships once a year with security updates in between ([Mozilla](https://mozilla.github.io/enterprise-admin-reference/guides/firefox-channels/)). Document which computers get which.
- [ ] **Install from one source.** Fleet offers [Firefox for macOS](https://fleetdm.com/software-catalog/firefox-darwin), [Firefox for Windows](https://fleetdm.com/software-catalog/firefox-windows), and [Firefox ESR](https://fleetdm.com/software-catalog/firefox@esr-darwin) as Fleet-maintained apps, with self-service for users.
- [ ] **Enforce a minimum version.** A [patch policy](https://fleetdm.com/guides/how-to-use-policies-for-patch-management-in-fleet) flags hosts behind the latest version and can install the update automatically (Fleet Premium). Test new releases on a pilot group first.
- [ ] **Verify extensions.** Mozilla's [ExtensionSettings policy](https://firefox-admin-docs.mozilla.org/reference/policies/extensionsettings/) blocks extensions and removes ones already installed. Fleet's agent reports what's installed through the [`firefox_addons` table](https://fleetdm.com/tables/firefox_addons), so you can confirm the block worked.
- [ ] **Confirm policies applied.** Fleet shows each [configuration profile](https://fleetdm.com/guides/custom-os-settings) as verified, pending, or failed per host.
- [ ] **Keep evidence from both layers.** Firefox Enterprise sends browser events to your SIEM. Fleet records version, profiles, and extensions per host. Managing Fleet as code in Git adds a reviewed history of every change ([Managing devices as code](https://fleetdm.com/whitepapers/modern-endpoint-management-managing-devices-as-code)).

On Linux, Fleet reports extensions and runs custom policies. Fleet doesn't offer Fleet-maintained apps or patch policies for Linux. You can install Firefox `.deb` and `.rpm` packages as custom packages instead. The [IT leader's guide to Linux device management](https://fleetdm.com/whitepapers/it-leaders-guide-to-linux-device-management) covers the rest of the Linux picture.

## Frequently asked questions

### Does Firefox Enterprise replace MDM?

No. Mozilla tells customers to keep their MDM. The console manages browser behavior, and device management installs, updates, and reports on Firefox.

### Is the Firefox Enterprise console open source?

Mozilla says the browser is open source but hasn't said whether the console is. Ask Mozilla if licensing matters for your evaluation.

### Can Fleet keep device data in Europe?

Yes. Fleet can host your instance in the EU, or you can run it in your own cloud, on-premises, or air-gapped. See [Fleet for Europe](https://fleetdm.com/europe).

## See it live

- **Get a demo** to see Firefox versions and extensions across your devices: [fleetdm.com/contact](https://fleetdm.com/contact)
- **See how Fleet keeps device data in Europe:** [Fleet for Europe](https://fleetdm.com/europe)

## More from Fleet

- [How to use policies for patch management in Fleet](https://fleetdm.com/guides/how-to-use-policies-for-patch-management-in-fleet)
- [Configuration profiles](https://fleetdm.com/guides/custom-os-settings)
- [Modern endpoint management: Managing devices as code](https://fleetdm.com/whitepapers/modern-endpoint-management-managing-devices-as-code) (whitepaper)
- [The IT leader's guide to Linux device management](https://fleetdm.com/whitepapers/it-leaders-guide-to-linux-device-management) (whitepaper)

## Sources

- Mozilla, [Introducing Firefox Enterprise](https://blog.mozilla.org/en/firefox/introducing-firefox-enterprise/).
- Mozilla, [Firefox Enterprise](https://www.firefox.com/enterprise/firefox-enterprise/).
- Mozilla, [Firefox for enterprise](https://www.firefox.com/browsers/enterprise).
- Mozilla enterprise admin reference, [Firefox and Firefox ESR](https://mozilla.github.io/enterprise-admin-reference/guides/firefox-channels/).
- Mozilla enterprise admin reference, [Getting started](https://mozilla.github.io/enterprise-admin-reference/guides/getting-started/).
- Mozilla, [ExtensionSettings policy reference](https://firefox-admin-docs.mozilla.org/reference/policies/extensionsettings/).
- The Next Web, [Mozilla launches Firefox Enterprise for companies in Europe](https://thenextweb.com/news/mozilla-launches-firefox-enterprise-for-companies-in-europ).
- It's FOSS, [Firefox Enterprise](https://itsfoss.com/news/firefox-enterprise/).
- CyberInsider, [Mozilla launches Firefox Enterprise with self-hosted browser management](https://cyberinsider.com/mozilla-launches-firefox-enterprise-with-self-hosted-browser-management/).

*Want to see which Firefox version and extensions are on every computer you manage? [Talk to Fleet](https://fleetdm.com/contact).*

<meta name="articleTitle" value="Firefox Enterprise and the managed browser checklist for IT">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-15">
<meta name="description" value="Firefox Enterprise governs the browser. Use this checklist to install, update, and audit Firefox on every computer you manage.">

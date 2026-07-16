# WWDC 2026: What IT admins need to know

The WWDC 2026 headlines are all Siri AI and a rebuilt Apple Intelligence stack. That stuff is interesting. It's also mostly someone else's problem.

For IT admins, WWDC 2026 is a migration year. Apple is removing legacy MDM mechanisms and replacing them with Declarative Device Management (DDM). Some of it is urgent. Some of it just needs a plan. Here's what matters and in what order.

## Legacy software update commands stop working in OS 27

This one requires action now.

In all OS 27 releases (iOS 27, iPadOS 27, macOS 27, watchOS 27, tvOS 27, visionOS 27), Apple is removing legacy software update MDM support. That means:

- Software update commands
- Software update queries
- Recommended cadence settings
- Software update deferrals and Background Security Improvements restrictions

These don't fail gracefully. They stop functioning entirely.

If your MDM still relies on legacy software update management, you lose update enforcement the moment a device upgrades to OS 27. That's not a deprecated feature with a grace period. It's gone.

What to do: verify your MDM vendor has shipped DDM-based software update management before the fall rollout. If you're using Fleet, declarative software update management is already available and this is exactly what it was built for. If you're not sure about your vendor, ask directly. Don't wait for a fall release to find out.

## TLS requirements are getting stricter

Apple is raising the transport security floor for system processes involved in device management: enrollment, profile installation, app installation, and software updates.

New requirement: TLS 1.2 minimum with ATS-compliant cipher suites and certificates. If your MDM or management infrastructure falls short, things break in ways that aren't always obvious. Enrollment failures, profiles not installing, update commands silently failing.

Apple published [support article 126655 ("Prepare your network environment for stricter security requirements")](https://support.apple.com/en-us/126655) to help you audit. Run it against your MDM and any internal management endpoints now, not in September. If you're using Fleet, you no action needed. Fleet meets the new requirements.

## New and migrated declarative configurations

> Enable the [`mdm.allow​_all​_declarations` feature flag](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-allow-all-declarations) to deploy any device-scoped, configuration [declaration (DDM profile)](https://developer.apple.com/documentation/devicemanagement/devicemanagement-declarations) with Fleet. Assets and user-scoped declarations are [coming in Fleet 4.90](https://github.com/fleetdm/fleet/issues/38986). At the same time, Fleet will enable this feature flag out-of-the-box.

Apple keeps expanding what DDM can express. Here's what's new in OS 27.

**VPN and network:** VPN configurations can now be delivered as declarative configurations, including IKEv2, IPSec, Always-On, DNS proxy, DNS settings, and relay. Credentials are deliverable as declarative assets with automated renewal. A meaningful step up from managing VPN profiles by hand.

**Intelligence, Siri, and keyboard controls:** The `com.apple.configuration.intelligence.settings`, `.external-intelligence.settings`, `.siri.settings`, and `.keyboard.settings` configurations move AI feature management into DDM. The legacy MDM restriction keys for these were deprecated in the 26.4 releases. If you've been putting that migration off, the runway is ending.

Notable controls: `AllowGenmoji`, `AllowImagePlayground`, `AllowWritingTools`, `AllowImageWand`, and per-app AI feature controls. Worth reviewing against your acceptable-use policies before OS 27 ships.

**Web content filter plugin:** A new `com.apple.configuration.webcontent-filter.plugin` declarative configuration.

**Content caching (macOS 27):** The `com.apple.configuration.content-cache.settings` configuration replaces the `com.apple.AssetCache.managed` profile. New status items for cache info, parents, and peers. Custom HTTPS reporting endpoints are supported. If you run content caches, plan the migration.

**Configuration profiles as declarative assets:** Legacy profiles can be delivered as declarative assets via the new `ProfileAssetReference` key in `com.apple.configuration.legacy`. Integrity verification is built in. This is a useful bridge for teams partway through the DDM transition.

## App management changes

**New unified app settings configuration:** `com.apple.configuration.app.settings` consolidates allowed and denied app management for iOS, iPadOS, tvOS, and visionOS. On macOS it adds binary-level control via Endpoint Security. A new `AlwaysAllowManagedApps` key preserves managed app access regardless of your allow-list configuration.

**Deprecation to act on:** The `com.apple.applicationaccess.new` profile is deprecated in macOS 27. If you're using it for app management on Mac, plan the migration now.

**Consolidated privacy consent:** A new `Privacy` key in `com.apple.configuration.app.settings` handles default app permissions in one place: camera, microphone, location, Bluetooth, local network, and more. The corresponding `com.apple.TCC.configuration-profile-policy` keys are deprecated. There's a matching `Privacy` key in `com.apple.configuration.safari.settings` for website camera and microphone defaults.

## Enrollment and identity changes

**Backup restoration no longer restores management state.** Starting in iOS 27, iPadOS 27, and visionOS 27, devices don't restore device management info (enrollment profile, management config, supervision status) from backup. Devices re-enroll via ADE after restore. The `do_not_use_profile_from_backup` key has no effect in OS 27. Update your device restoration and re-enrollment runbooks accordingly.

**Extensible SSO and Platform SSO come to DDM.** A new `com.apple.configuration.extensible-sso` declarative configuration brings both to the DDM model. Platform SSO on macOS 27 adds web-based authentication: IdP web view at the FileVault unlock screen, lock screen, and login window, plus QR-code sign-in and an offline grace period.

**EWS is going away, in phases.** Microsoft begins disabling Exchange Web Services on October 1, 2026, with full permanent shutdown on April 1, 2027. Apple is working with Microsoft to move Mail, Calendar, Contacts, Notes, and Reminders to the Microsoft Graph API. Apple's own Graph API support arrives in a future macOS 27 update, not at general availability. If your organization uses native Apple apps with Exchange, plan ahead: the October date is the start of the disablement window, not the final cutoff.

**Return to Service improvements:** The new `ShouldRetryEnrollment` key enables automatic enrollment retry with backoff up to five minutes. Language and region can now be set via `language` and `region` keys.

## New status items and observability

Worth knowing about for monitoring and compliance:

- `device.system.health`: reports on hardware component genuineness (baseband, camera, Face ID, Touch ID, NFC, UWB). iPhone and iPad only. Useful for detecting device tampering.
- `security.lockdown-mode`: reports whether Lockdown Mode is active on a supervised device.
- `mdm.enrollment-type`, `mdm.is-awaiting-configuration`, `mdm.is-return-to-service`, `mdm.is-shared-ipad`, `mdm.push-magic`, `mdm.push-token`: more granular enrollment state visibility.

**AppleCare remote log collection:** Two new MDM commands, `TriggerEnhancedLogCollection` and `CancelEnhancedLogCollection`, enable remote log collection on supervised devices for AppleCare support cases. Apple's documentation specifies an AppleCare Enterprise agreement is required to test this feature in beta releases.

## Intel Mac support timeline

macOS 26 (Tahoe) was the last release with full Intel Mac support. macOS 27 (Golden Gate) is Apple Silicon only.

Apple will provide three more years of security updates for Intel Macs, putting the end of that window at roughly fall 2028. Rosetta continues through macOS 27, and there's a new `allowRosettaUsageAwareness` MDM key to suppress the deprecation notice for users.

If your fleet still includes Intel hardware, the refresh conversation with leadership and procurement needs to start now. Three years sounds like runway until you're doing it all at once.

## What the AI changes mean for policy

Siri AI (Apple's rebuilt, agentic Siri) and expanded Apple Intelligence features are the consumer headline. The practical questions for IT are narrower.

Use the new `com.apple.configuration.intelligence.settings` and `.siri.settings` declarative configurations to control what AI features are available on managed devices. Specific controls cover Writing Tools, Image Playground, Genmoji, Image Wand, and per-app feature access.

A few things worth setting user expectations on: Siri AI requires a user waitlist, launches in English only, and isn't available in the EU on iPhone or iPad, or in China. The most capable on-device AI model requires 12GB of unified memory (iPhone 17 Pro or Air, M3+ Mac with 12GB, M4+ iPad with 12GB). Devices with less memory fall back to Private Cloud Compute.

## Suggested priorities

If you're building your OS 27 response plan:

1. Verify your MDM vendor's DDM software update support. This is the fire drill. Do it today. If you use Fleet, you're already set.
2. Audit TLS/ATS compliance across your MDM and management infrastructure using Apple's support article 126655. If you use Fleet, you're already set.
3. Migrate Intelligence, Siri, and keyboard restrictions to the new declarative configurations. Fleet already supports all declarations that are replacing deprecated v1 profiles (.mobileconfig). Assets and user-scoped declarations are [coming in Fleet 4.90](https://github.com/fleetdm/fleet/issues/38986).
4. Plan the app management migration away from `com.apple.applicationaccess.new` on macOS.
5. Update your re-enrollment runbooks to account for the backup restoration change.
6. Start the Intel Mac refresh conversation if you haven't already.

Developer betas are available today. Public betas land in July. Fall release is September. If you're starting DDM migration from scratch, there's not much runway.

_All configuration keys and features discussed here are pre-release. Apple flags that specifics may change before fall general availability, and some intelligence controls have documented enforcement gaps in early 27.0 seeds. Verify against Apple's deployment documentation as betas progress._

<meta name="articleTitle" value="WWDC 2026: What IT admins need to know">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-06-09">
<meta name="category" value="guides">
<meta name="description" value="WWDC 2026 delivers sweeping device management changes. Here's what IT admins need to prioritize before OS 27 ships this fall.">

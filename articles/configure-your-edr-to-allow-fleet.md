# Configure your EDR to allow Fleet

If your EDR flags Fleet, configure an allowlist or exclusion for Fleet's trusted binaries. Work with your EDR vendor to choose the narrowest option that fits your security and release process.

## How to configure your EDR

Most enterprise EDRs support allowlisting by publisher, binary path, or SHA-256 hash. Exact names and settings vary by vendor.

To get started:

1. Review the alert and confirm it came from Fleet activity on a managed host.
2. Decide whether your team wants to allowlist Fleet by publisher or path, or by specific binary hash.
3. Open a support case with your EDR vendor and ask for their recommended approach for asset visibility, device management, or telemetry collection tools.
4. Gather the Fleet signing and binary details your EDR vendor requires.
5. Test the allowlist in a small group before broad rollout.
6. Monitor for additional alerts after the change.

## Choose an allowlisting approach

### Option 1: Allowlist by publisher or binary path

Configure your EDR to trust binaries signed by Fleet's publisher, or exclude Fleet's binary paths from detection.

- Upside: Easy to configure and maintain.
- Downside: Current and future Fleet binaries are trusted automatically.

Most enterprise EDRs support this option through exclusions, exceptions, or trusted application settings.

### Option 2: Allowlist by SHA-256 hash

Allowlist only the exact Fleet binaries your team plans to deploy. This gives you tighter control, but it adds operational overhead for every release.

A practical approach is to run a small canary group on Fleet's edge channel before wider deployment. That lets your team review the specific binaries that will ship, allowlist the approved hashes in your EDR, and then promote the release more broadly.

A canary setup can pin only osquery to edge while keeping Orbit and Fleet Desktop on stable:

```yaml
agent_options:
  update_channels:
    orbit: stable
    osqueryd: edge
    desktop: stable
```

See the [agent configuration docs](https://fleetdm.com/docs/configuration/agent-configuration#update-channels) for details about update channels and rollback behavior.

## Contact Fleet and your EDR vendor

Your EDR vendor can tell you which allowlisting method they recommend and where to configure it.

Fleet can provide the technical details your vendor may request, such as developer ID, current binary hashes, signing details, and behavior documentation.

## Why your EDR may flag Fleet

Fleet collects host telemetry to help your team understand system activity. Some of the same behaviors used for visibility, such as inspecting processes, binaries, or system state, can overlap with behaviors that EDR tools monitor closely.

That overlap does not mean Fleet is acting maliciously. It means your EDR is detecting behavior that resembles activity it is designed to inspect.

## What the alert means

An alert on Fleet does not automatically mean Fleet is unsafe. Fleet is code-signed, distributed through TUF, and developed in the open at [github.com/fleetdm/fleet](https://github.com/fleetdm/fleet).

Allowlisting should still be a deliberate security decision. Choose the scope that matches your team's risk tolerance and operational model.

<meta name="articleTitle" value="Configure your EDR to allow Fleet">
<meta name="authorFullName" value="Dhruv Majumdar">
<meta name="authorGitHubUsername" value="karmine05">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-05-07">
<meta name="description" value="How to configure your EDR to allow Fleet, plus why EDR tools flag Fleet and what those alerts mean.">
<meta name="useBasicArticleTemplate" value="true">
<meta name="cardTitleForCustomersPage" value="Configure your EDR to allow Fleet">

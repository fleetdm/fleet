# Configure your EDR to allow Fleet

If your EDR flags Fleet, configure an allowlist or exclusion for Fleet's trusted binaries. Work with your EDR vendor to choose the narrowest option that fits your security and release process.

## How to configure your EDR

Most enterprise EDRs support allowlisting by publisher, binary path, or SHA-256 hash. Exact names and settings vary by vendor.

To get started:

1. Review the alert and confirm it came from Fleet activity on a managed host.
2. Check which process was flagged. If it's a Fleet binary (for example, `orbit.exe`), an allowlist by publisher, path, or hash usually resolves it. If it's a process Fleet launched (for example, `powershell.exe` with `orbit.exe` as its parent), you'll likely need a [behavioral (IOA) exclusion](#behavioral-ioa-exclusions).
3. Decide whether your team wants to allowlist Fleet by publisher or path, or by specific binary hash.
4. Open a support case with your EDR vendor and ask for their recommended approach for asset visibility, device management, or telemetry collection tools.
5. Gather the Fleet signing and binary details your EDR vendor requires.
6. Test the allowlist in a small group before broad rollout.
7. Monitor for additional alerts after the change.

## Choose an allowlisting approach

### Option 1: Allowlist by publisher or binary path

Configure your EDR to trust binaries signed by Fleet's publisher, or exclude Fleet's binary paths from detection.

- Upside: Easy to configure and maintain.
- Downside: Current and future Fleet binaries are trusted automatically.

Most enterprise EDRs support this option through exclusions, exceptions, or trusted application settings.

**Publisher**: Add `Fleet Device Management Inc` as an exclusion rule.

**Binary paths**: Use a recursive path exclusion for:
- macOS/Linux: `/opt/orbit/`
- Windows: `C:\Program Files\Orbit\`

> **Note:** Publisher and path exclusions only apply when a Fleet binary is the flagged process. They don't cover behavioral detections on processes that Fleet launches, such as PowerShell running a software install script. If alerts continue after you configure this option, see [Behavioral (IOA) exclusions](#behavioral-ioa-exclusions).

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

## Behavioral (IOA) exclusions

Some EDRs flag Fleet based on behavior rather than on the binary itself. For example, CrowdStrike Falcon uses indicators of attack (IOAs) to detect process chains that resemble an attacker using a software deployment tool to run code.

When Fleet runs a script (such as a software install, post-install, or custom script), Orbit launches PowerShell on Windows as a child process. If your EDR flags that child process, the detection is on `powershell.exe`, not on Orbit. Publisher and path exclusions for Fleet won't apply, so you need a behavioral exclusion instead.

You may need a behavioral exclusion if:

- Detections continue after you allowlist Fleet by publisher or path.
- The flagged process is `powershell.exe` and its parent process is `orbit.exe`.
- Alerts spike when Fleet installs software or when running scripts.

Other EDRs offer similar behavioral exclusions under different names. Ask your EDR vendor where to configure them.

### Example: CrowdStrike Falcon IOA exclusion for Fleet scripts (Windows)

In one CrowdStrike deployment, Falcon flagged Fleet's software install scripts with the `BundledMalware` IOA (tactic: Execution, technique: Software Deployment Tools). This was then resolved with an IOA exclusion scoped to PowerShell processes launched by Orbit to run Fleet's scripts.

The simplest way to create the exclusion is from the detection in the Falcon console, which pre-fills the fields. Then narrow each field to match Fleet's process chain:

```
IOA name:                   BundledMalware  (use the IOA name from your detection)
Image filename:             .*\\Windows\\System32\\WindowsPowerShell\\v1\.0\\powershell\.exe
Command line:               powershell\s+-MTA\s+-ExecutionPolicy\s+Bypass\s+-File\s+.*\\WINDOWS\\SystemTemp\\.*\\(?:install-script|post-install-script|rollback-script|script)\.ps1
Parent image filename:      .*\\Program\s+Files\\Orbit\\bin\\orbit\\orbit\.exe
Parent command line:        ".*\\Program\s+Files\\Orbit\\bin\\orbit\\orbit\.exe"
Grandparent image filename: .*\\Windows\\System32\\services\.exe
Grandparent command line:   .*\\WINDOWS\\system32\\services\.exe
```

Tips:

- Keep the exclusion as specific as possible. The example above matches on the IOA, the PowerShell command line, and Orbit as the parent process, rather than excluding PowerShell broadly.
- Each IOA exclusion applies to a specific IOA pattern. If more than one IOA fires on Fleet activity, you may need an exclusion for each.
- Copy the script path from your own detection's command line. On Windows, processes running as SYSTEM may use `C:\Windows\SystemTemp` for temporary files.
- Before saving, review the list of recent detections that Falcon shows the exclusion would have covered. Exclusions can take up to 40 minutes to take effect.
- Scope the exclusion to a small host group first, then expand once detections stop.

## Contact Fleet and your EDR vendor

Your EDR vendor can tell you which allowlisting method they recommend and where to configure it. If you notice a new flag against the orbit binary, please contact your EDR vendor support team to report the false positive; they will let you know the best path forward to address any exceptions you may want to make.

Fleet is in active communication with EDR vendors to resolve false-positive flagging of the fleetd agent. Fleet can also provide the technical details your vendor may request, such as developer ID, current binary hashes, signing details, and behavior documentation.

## Why your EDR may flag Fleet

Fleet collects host telemetry to help your team understand system activity. Some of the same behaviors used for visibility, such as inspecting processes, binaries, or system state, can overlap with behaviors that EDR tools monitor closely.

That overlap does not mean Fleet is acting maliciously, it means your EDR is detecting behavior that resembles activity it is designed to inspect. This is a known false-positive scenario that can occur when changes to the agent's behavior trigger heuristic-based detections, and it's especially common after a Fleet update.

It's common for security products to be falsely flagged as malicious because they need to access security-sensitive data (keychains, certificates, system configurations) to do their intended work. This is a known pattern across the industry and is not unique to Fleet; endpoint agents from many vendors encounter the same heuristic-based false positives.

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

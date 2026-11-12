# Windows 11 Enterprise benchmarks (Intune)

Fleet's policies have been written against v8.1 of the CIS benchmark for Windows 11 Enterprise, targeting devices enrolled in Microsoft Intune. You can refer to the [CIS website](https://www.cisecurity.org/cis-benchmarks) for full details about this version.

For requirements and usage details, see the [CIS Benchmarks](https://fleetdm.com/docs/using-fleet/cis-benchmarks) documentation.

### Contents

| Folder | Description |
|--------|-------------|
| `policies/` | GitOps-compatible policy YAML (bl/l1/l2) — import via `fleetctl apply` or reference with `- path:` in `fleet.yml` |
| `configuration-profiles/` | SyncML XML profiles — upload via Fleet UI or `fleetctl apply` to enforce the settings checked by the policies |
| `scripts/` | PowerShell scripts — upload via Fleet UI or `fleetctl apply` and link as `run_script` remediation in the corresponding policy |

### Policy files

Policies are split by CIS level:

| File | Level | Description |
|------|-------|-------------|
| `bl_win11_intune.yml` | Baseline | Foundational controls required regardless of level |
| `l1_win11_intune.yml` | Level 1 | Standard security configuration for most environments |
| `l2_win11_intune.yml` | Level 2 | High-security configuration for environments requiring stricter controls |

### Limitations

> None. All items in this version of the benchmark are able to be automated.

### How these policies work

Each policy uses `mdm_bridge` queries to read CSP (Configuration Service Provider) values via OMA-URI, allowing Fleet to verify settings that were applied through Intune configuration profiles.

# ADR-0009: Deprecate nvd feed

## Status

Pending

## Date

2026-02-19

## Context

NVD stopped enriching CVEs with CPE data starting February 1, 2024
April 2024 - PR #18168
PR introduced VulnCheck api to "replace existing entries only if the NVD CVE does not have them"
December 2024 - PR #24318
Now processes all VulnCheck data, not just post-Feb 2024

The proposal here is to switch over to exclusively using the vulncheck api rather than a combination of NVD & vulncheck. If NVD will no longer enrich their data, there is no reason to continue pulling NVD data. The Vulncheck feed has data on CVEs dating back to 2002.

Dual sync paths:
`sync()` method handles NVD API 2.0 synchronization
`DoVulnCheck()` method handles VulnCheck archive downloads
`processVulnCheckFile()` method processes VulnCheck data and merges with existing NVD data

We stores CVE data in NVD's legacy 1.1 JSON format because the vulnerability matching library (github.com/facebookincubator/nvdtools) doesn't support the new API 2.0 format
Both NVD and VulnCheck data are converted to this legacy format before being stored

## Changes

Eliminate complex merge logic in cve_syncer.go:`updateVulnCheckYearFile`. No more tracking which CVEs came from which source and no more conflict resolution between dual sources. 

### Phased rollout

### 1:  Feature flag implementation
When enabled, skip NVD API synchronization entirely
Use only VulnCheck data for building the CVE feed
Rebuild the complete CVE database from 2002 to present using VulnCheck-only mode
Compare CVE counts and CPE configuration coverage between dual-source and VulnCheck-only modes
Validate that vulnerability detection accuracy matches or exceeds current implementation

### 2: Default cutover
Switch default behavior to VulnCheck-only
Deprecate NVD sync code paths but keep them available behind the feature flag

### 3: Delete NVD
Remove NVD sync code from [cve_syncer.go]
Remove dual-source merge logic in `updateVulnCheckYearFile()`
Simplify to a single data source architecture
Remove feature flag
Update documentation

# Consequences
## Positive
* Eliminate ~300 lines of merge logic
* Single source of truth
* Better data quality
* Reduced API dependencies: No longer dependent on NVD API rate limits (currently 6 seconds between requests per)
* Lower maintenance burden
* Faster sync times

## Negative
* Single vendor dependency
* Migration effort


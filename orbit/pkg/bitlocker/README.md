# BitLocker (Disk encryption on Windows)

BitLocker is Windows' disk encryption feature. See
https://learn.microsoft.com/en-us/windows/security/operating-system-security/data-protection/bitlocker/
for more background information

When you enable disk encryption in Fleet, BitLocker is used to accomplish the encryption.

```mermaid
---
title: BitLocker flow in Fleet
---
sequenceDiagram
participant Orbit
link Orbit: Dashboard @ https://dashboard.contoso.com/alice
    loop Get Orbit config (every 30s by default)
        Orbit<<-->>Fleet: Get Orbit config notification (`POST /api/fleet/orbit/config`)
    Orbit-)Fleet: Disk encryption status
    end
```
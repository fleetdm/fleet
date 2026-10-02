# WiX custom-action DLLs

These DLLs are third-party binaries from the [WiX Toolset](https://github.com/wixtoolset/wix3) v3.14.1. They are licensed under the Microsoft Reciprocal License (MS-RL), see [LICENSE.TXT](LICENSE.TXT). They are not covered by Fleet's license.

The pure Go MSI writer (`orbit/pkg/packaging/msi`) embeds them, unmodified, into the `Binary` table of every fleetd MSI, exactly as WiX did. They provide the `WixQuietExec64` custom action (runs `installer_utils.ps1` without a console window) and the `ServiceConfig` actions (restart the `Fleet osquery` service on failure).

| File | Architecture | Version | SHA-256 |
|------|--------------|---------|---------|
| `WixCA.dll` | x86 (used by x64 and arm64 MSIs for `WixQuietExec64`) | 3.14.1.8722 | `78860e15e474cc2af7ad6e499a8971b6b8197afb8e49a1b9eaaa392e4378f3a5` |
| `WixCA_A64.dll` | arm64 (`ServiceConfig` actions in arm64 MSIs) | 3.14.1.8722 | `d7c582d6707918dea632a120926f2be02b56a0d766e9f40b79aa8555287b7b10` |

## Source

The DLLs are built from `src/ext/ca/wixca/dll` at tag [`wix3141rtm`](https://github.com/wixtoolset/wix3/tree/wix3141rtm/src/ext/ca/wixca/dll) of `wixtoolset/wix3`. Both are Authenticode-signed by the .NET Foundation.

They were extracted verbatim from the `Binary.WixCA` and `Binary.WixCA_A64` streams of fleetd MSIs built with WiX 3.14.1 (the `fleetdm/wix` image). To verify, build an MSI with a fleetctl that uses WiX and compare:

```sh
msidump -s -d out fleet-osquery.msi
shasum -a 256 out/_Streams/Binary.WixCA*
```

Don't modify or rebuild these files. MS-RL requires distributing the source of any modified file, and the MSI's byte-for-byte equivalence with WiX output (see `tools/msi-compare`) depends on them.

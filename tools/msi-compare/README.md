# msi-compare

Validates that two fleetd Windows installers (`.msi`) are equivalent byte for
byte, except for build-time nondeterminism. Built to validate the pure Go MSI
writer (`orbit/pkg/packaging/msi`) against WiX v3 output during the removal
of the `fleetdm/wix` Docker dependency
([#50503](https://github.com/fleetdm/fleet/issues/50503)).

Two MSIs built from identical inputs are never bit-identical: every build
generates a fresh `ProductCode`, `PackageCode`, and per-component GUIDs,
stamps timestamps in the summary-information stream and CAB entries, and the
embedded CAB's compressed bytes depend on the deflate implementation. The
script normalizes exactly that and requires everything else to match byte for
byte: table list, stream list, every table dump, summary information, Binary
table streams (custom-action DLLs), CAB structure (entry order/names/sizes),
and every extracted payload file.

## Requirements

```sh
brew install msitools cabextract
```

## Usage

```sh
# reference: built by a released fleetctl (WiX); candidate: built by the Go writer
./compare-msi.py reference.msi candidate.msi

# keep the work dir (table dumps, extracted payloads) for inspection
./compare-msi.py --keep reference.msi candidate.msi
```

Exits 0 when all checks pass. Self-test: comparing two consecutive builds
made by the *same* toolchain from the same inputs must pass; if it does not,
the normalization (not the installers) is broken.

Note: the compared payloads include files whose content depends on the
fleetctl build (e.g. `certs.pem`, TUF targets pinned at download time), so
compare builds made from the same fleetctl code/certs, or expect payload
diffs that have nothing to do with the MSI writer.

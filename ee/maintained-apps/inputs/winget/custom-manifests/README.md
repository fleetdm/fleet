# Fleet custom winget manifests

This directory holds hand-maintained winget-schema manifests for Windows
Fleet-maintained apps whose installer is not published to
`microsoft/winget-pkgs`, so the ingester can't fetch it from there. It is the
Windows counterpart of [`../../homebrew/custom-tap/`](../../homebrew/custom-tap/).

```
custom-manifests/
├── README.md
└── <PackageIdentifier>/
    ├── <PackageIdentifier>.installer.yaml      # version, URL, sha256, UpgradeCode
    └── <PackageIdentifier>.locale.en-US.yaml   # Publisher, PackageName
```

## How it hooks into the FMA ingester

The app's input manifest one directory up (`../<app>.json`) sets
`manifest_path` to this app's directory. When `go run cmd/maintained-apps/main.go`
runs, the ingester reads the two YAML files from disk and skips the GitHub
lookup. The `PackageIdentifier` inside the files must match the input's
`package_identifier`; pick one that can't collide with a real winget package
(e.g. `Druva.inSync.GovCloud`).

## Adding an app

1. Download the installer and read its identity with `msiinfo export <file>.msi Property`
   (`brew install msitools`): `ProductVersion`, `Manufacturer`, `UpgradeCode`.
2. Write `<PackageIdentifier>/<PackageIdentifier>.installer.yaml` and
   `.locale.en-US.yaml`. Use an existing directory here as the template. Only the
   fields the ingester reads are needed; `ProductCode` can be omitted for
   machine-scope MSIs because Fleet uninstalls by `UpgradeCode`.
3. Create `../<app>.json` pointing `manifest_path` at the directory, then run
   `go run cmd/maintained-apps/main.go --slug="<app>/windows"` from the repo root.
4. Follow the rest of the FMA contributor flow in [`../../../README.md`](../../../README.md).

## Updating an app

Bump `PackageVersion`, `InstallerUrl` and `InstallerSha256` together (winget
convention is uppercase hex; the ingester lowercases it), regenerate the output
manifest, and commit the YAML and `outputs/<app>/windows.json` together.

## Apps

| Directory | Upstream source |
|---|---|
| `Druva.inSync.GovCloud` | The GovCloud Windows entry of `https://downloads.druva.com/insync/js/data.json` (the feed behind the "inSync GovCloud" selection on https://downloads.druva.com/insync/). Trails the main-cloud release by one version. |

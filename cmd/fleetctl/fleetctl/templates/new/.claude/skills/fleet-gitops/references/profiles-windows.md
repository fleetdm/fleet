# Windows profiles (CSP SyncML)

A Windows profile is a fragment of OMA-DM SyncML: one or more top-level `<Replace>` (or `<Add>`, `<Exec>`, `<Atomic>`) commands, no XML declaration and no root element (Fleet wraps it). Each command sets one configuration service provider (CSP) node. List the file under `controls.windows_settings.configuration_profiles`; the file name (without `.xml`) is the profile name Fleet shows and must be unique within the fleet across all profile types.

```xml
<Replace>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength</LocURI>
    </Target>
    <Data>12</Data>
  </Item>
</Replace>
```

- `LocURI` must start with `./` (`./Device/...` for device scope, `./User/...` for the enrolled user) and never contain `../`. Policy CSP nodes are `./Device/Vendor/MSFT/Policy/Config/<Area>/<Policy>`; dedicated CSPs are `./Device/Vendor/MSFT/<CSP>/...`.
- `<Format>` must match the node's DDF `DFFormat`: `int`, `chr`, `bool`, `b64`, `node`. A mismatch deploys and then fails on the device.
- Keep one topic per file and use `<!-- comments -->` to say what each node does; reviewers read these.
- Fleet owns disk encryption (`/Vendor/MSFT/BitLocker`), Windows Update policy (`/Vendor/MSFT/Policy/Config/Update`, use `windows_updates`), and remote wipe; a profile that targets those nodes is rejected.

## Keys come from the DDF, not from memory

Microsoft publishes every CSP's schema as a Device Description Framework (DDF) file, the same source Intune's settings catalog is built from. Before writing a node, fetch its DDF page and read the node's `<DFProperties>`:

```
https://learn.microsoft.com/en-us/windows/client-management/mdm/<csp>-ddf-file     # e.g. policy-ddf-file, firewall-ddf-file, devicelock-ddf-file
https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-<area>  # prose per Policy CSP area, e.g. policy-csp-devicelock
https://learn.microsoft.com/en-us/windows/client-management/mdm/<csp>-csp          # prose per dedicated CSP
```

What to confirm for each node you set:

| DDF field | What it tells you |
|---|---|
| `AccessType` | Whether the node accepts `Replace`/`Add` at all; around five hundred nodes are `Get`-only and a `Replace` to them deploys, then fails on the device |
| `DFFormat` | The `<Format>` to use |
| `MSFT:AllowedValues` | The enum, range, or regex the node accepts; copy values exactly, including the ones that read backwards (`DeviceLock/DevicePasswordEnabled` is `0` for enabled) |
| `MSFT:Applicability` | Minimum OS build and allowed editions; a valid node on the wrong SKU is a silent no-op |
| `MSFT:DependencyBehavior` | Other nodes that must be set for this one to take effect (`MinDevicePasswordLength` does nothing without `DevicePasswordEnabled`) |
| `MSFT:GpMapping` / `MSFT:AdmxBacked` | The Group Policy name, and for ADMX-backed policies the `.admx` file and area path you need to build the `<Data>` payload |

ADMX-backed policies take an XML string as data (`<![CDATA[<enabled/><data id="..." value="..."/>]]>`); the element ids and accepted values come from the `.admx` file named in the DDF. https://fleetdm.com/guides/creating-windows-csps walks through assembling one.

Quote the DDF node, its `AllowedValues`, and its applicability in your report so the reviewer can check the one thing static validation can't: that the value is the one you meant.

## Validate

```bash
{ echo '<r>'; cat "path/to/profile.xml"; echo '</r>'; } | xmllint --noout -   # well-formed (multiple top-level elements need the wrapper)
python3 <skill>/scripts/validate.py                                           # top-level elements, LocURI shape, Format present, reserved nodes, name conflicts
```

If the repo keeps a copy of the DDF files (for example under `platforms/windows/schema/`), check every node name exists:

```bash
grep -oh '<LocURI>[^<]*</LocURI>' path/to/profile.xml | sed 's|</*LocURI>||g' | while read -r uri; do
  grep -rqs "<NodeName>${uri##*/}</NodeName>" platforms/windows/schema/ || echo "unknown node: $uri"
done
```

(The validator does this automatically with `--ddf-dir <dir>`.) Then the dry run: Fleet checks well-formedness, the top-level element, LocURI format, reserved nodes, and name conflicts. Anything about values, formats, dependencies, or applicability stays a review-time check against the DDF.

## Verify on a device

Hosts > a host > OS settings shows Verified or Failed. Policy CSP settings land in the registry under `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\PolicyManager\current\device\<Area>`; confirm the value with a live query:

```sql
SELECT path, data FROM registry
WHERE path LIKE 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\PolicyManager\current\device\DeviceLock\%';
```

Dedicated CSPs write elsewhere (for example `PolicyManager\Providers`), so check the path on one pilot host before building a policy around it.

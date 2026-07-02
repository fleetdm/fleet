# CPE Translations

CPE Translations are rules to address bugs when translating Fleet software to Common Platform Enumerations
(CPEs) which are used to identify software in the National Vulnerability Database (NVD)

To improve accuracy when [mapping software to CVEs](../../../docs/Using%20Fleet/Vulnerability-Processing.md), we can add data to [cpe_translations.json](./cpe_translations.json)

## How CPE translations work

CPE Translations are defined in `cpe_translations.json` and currently released in
[GitHub](https://github.com/fleetdm/nvd) once a day.  The rules are specified in JSON format and
and each rule consists of a `software` and a `filter` object.

`software` defines matching logic on what Fleet Software this rule should apply to.  You can use one
or more of the below attributes to match on.  Each attribute is an array of string or regex
matches (a regex string is identified by a leading and trailing `/`).  
A match on the attribute is found if at least 1 item in the array matches.  If multiple
attributes are defined, then a match is needed for each attribute.  (ie. `name == Zoom.app` &&
`source == apps`)

`software` attributes:

- `name`: A software name attribute 
- `bundle_identifier`: A software bundle_identifier attribute (macOS only)
- `source`: A software source attribute (ie. `apps`, `chrome_extensions`, etc...)

**example**: Search Fleet software for items that match: (bundle_identifier == us.zoom.xos) AND (source = apps)

```json
"software": {
      "bundle_identifier": ["us.zoom.xos"],
      "source": ["apps"]
    }
```

If the software rule matches, then Fleet will search known NVD CPEs (stored in a local sqlite database) using the
specified filters or skip the software item based on the `filter` specified.  

`filter` attributes:

- `product`: array of strings to search by product field.  If not specified, the software name is used.
- `vendor`: array of strings to search by vendor field
- `target_sw`: array of strings to search by target_sw field
- `part`: string to override the default "a" Part value
- `skip`: boolean; software is skipped if `true`.  This overrides any other filters set.

Like the software matching logic, filter items are matched by OR within the array, and AND between
filter items

**example**: Query the CPE database for a CPE that matches:
(product == zoom OR product == meetings) AND (vendor == zoom) AND (target == macos OR target == mac_os)

```json
"filter": {
      "product": ["zoom", "meetings"],
      "vendor": ["zoom"],
      "target_sw": ["macos", "mac_os"]
    }
```



# CVE resolved versions

Some NVD records describe an affected version range using only `versionEndIncluding` (the last
vulnerable version) and omit `versionEndExcluding` (the first fixed version). When that happens,
Fleet cannot derive `resolved_in_version` from the feed, so the fixed version is reported as empty
even when it is publicly known.

To work around this, Fleet consults a resolved-versions override in
[cve_resolved_versions.json](./cve_resolved_versions.json). Like CPE translations, it is released in
[GitHub](https://github.com/fleetdm/nvd) and downloaded once a day, so entries can be added or
removed without cutting a Fleet release. It is only used as a fallback: authoritative NVD
`versionEndExcluding` data always takes precedence, and entries should be removed once NVD publishes
the correct data.

The file is a JSON array of overrides. Each override matches a single CVE against a `vendor` and
`product` and supplies the upstream fix version:

```json
[
  {
    "cve": "CVE-2025-63389",
    "vendor": "ollama",
    "product": "ollama",
    "resolved_in_version": "0.12.4"
  }
]
```

An override is applied only to hosts that are (a) already matched as affected by the CVE, (b) whose
software `vendor`/`product` match the override, and (c) whose version is below `resolved_in_version`.

## Testing CVE resolved versions (end-to-end)

Follow the same steps as [Testing CPE Translations](#testing-cpe-translations-end-to-end), but host
and point Fleet at `cve_resolved_versions.json` instead:

```bash
FLEET_VULNERABILITIES_CVE_RESOLVED_VERSIONS_URL="http://localhost:8082/cve_resolved_versions.json" ./build/fleet serve --dev --dev_license --logging_debug
```

## Testing CPE Translations (end-to-end)

1. make the [appropriate](../../../docs/Using%20Fleet/Vulnerability-Processing.md#Improving-accuracy) changes to cpe_translations

2. host this file on a local web server

    ```bash
    go run ./tools/file-server/main.go 8082 ./server/vulnerabilities/nvd/
    ```

3. (re)launch your local fleet server with one of the following

    Config method

    ```yaml
    vulnerabilities:
    cpe_translations_url: "http://localhost:8082/cpe_translations.json"
    ```

    Environment method

    ```bash
    FLEET_VULNERABILITIES_CPE_TRANSLATIONS_URL="http://localhost:8082/cpe_translations.json" ./build/fleet serve --dev --dev_license --logging_debug
    ```

4. trigger a vulnerabilities scan

    ```bash
    fleetctl trigger --name vulnerabilities
    ```

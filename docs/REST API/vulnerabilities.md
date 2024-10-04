# Vulnerabilities

## List vulnerabilities

Retrieves a list of all CVEs affecting software and/or OS versions.

`GET /api/v1/fleet/vulnerabilities`

### Parameters

| Name                | Type     | In    | Description                                                                                                                          |
| ---      | ---      | ---   | ---                                                                                                                                  |
| team_id             | integer | query | _Available in Fleet Premium_. Filters only include vulnerabilities affecting the specified team. Use `0` to filter by hosts assigned to "No team".  |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                       |
| per_page                | integer | query | Results per page.                                                                                                                                                          |
| order_key               | string  | query | What to order results by. Allowed fields are: `cve`, `cvss_score`, `epss_probability`, `cve_published`, `created_at`, and `host_count`. Default is `created_at` (descending).      |
| order_direction | string | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| query | string | query | Search query keywords. Searchable fields include `cve`. |
| exploit | boolean | query | _Available in Fleet Premium_. If `true`, filters to only include vulnerabilities that have been actively exploited in the wild (`cisa_known_exploit: true`). Otherwise, includes vulnerabilities with any `cisa_known_exploit` value.  |


#### Default response

`Status: 200`

```json
{
  "vulnerabilities": [
    {
      "cve": "CVE-2022-30190",
      "created_at": "2022-06-01T00:15:00Z",
      "hosts_count": 1234,
      "hosts_count_updated_at": "2023-12-20T15:23:57Z",
      "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2022-30190",
      "cvss_score": 7.8,// Available in Fleet Premium
      "epss_probability": 0.9729,// Available in Fleet Premium
      "cisa_known_exploit": false,// Available in Fleet Premium
      "cve_published": "2022-06-01T00:15:00Z",// Available in Fleet Premium
      "cve_description": "Microsoft Windows Support Diagnostic Tool (MSDT) Remote Code Execution Vulnerability.",// Available in Fleet Premium
    }
  ],
  "count": 123,
  "counts_updated_at": "2024-02-02T16:40:37Z",
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```


## Get vulnerability

Retrieve details about a vulnerability and its affected software and OS versions.

If no vulnerable OS versions or software were found, but Fleet is aware of the vulnerability, a 204 status code is returned.

### Parameters

| Name    | Type    | In    | Description                                                                                                                  |
|---------|---------|-------|------------------------------------------------------------------------------------------------------------------------------|
| cve     | string  | path  | The cve to get information about (format must be CVE-YYYY-<4 or more digits>, case-insensitive).                             |
| team_id | integer | query | _Available in Fleet Premium_. Filters response data to the specified team. Use `0` to filter by hosts assigned to "No team". |

`GET /api/v1/fleet/vulnerabilities/:cve`

### Example

`GET /api/v1/fleet/vulnerabilities/cve-2022-30190`

#### Default response

`Status: 200`

```json
"vulnerability": {
  "cve": "CVE-2022-30190",
  "created_at": "2022-06-01T00:15:00Z",
  "hosts_count": 1234,
  "hosts_count_updated_at": "2023-12-20T15:23:57Z",
  "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2022-30190",
  "cvss_score": 7.8,// Available in Fleet Premium
  "epss_probability": 0.9729,// Available in Fleet Premium
  "cisa_known_exploit": false,// Available in Fleet Premium
  "cve_published": "2022-06-01T00:15:00Z",// Available in Fleet Premium
  "cve_description": "Microsoft Windows Support Diagnostic Tool (MSDT) Remote Code Execution Vulnerability.",// Available in Fleet Premium
  "os_versions" : [
    {
      "os_version_id": 6,
      "hosts_count": 200,
      "name": "macOS 14.1.2",
      "name_only": "macOS",
      "version": "14.1.2",

      "resolved_in_version": "14.2",
      "generated_cpes": [
        "cpe:2.3:o:apple:macos:*:*:*:*:*:14.2:*:*",
        "cpe:2.3:o:apple:mac_os_x:*:*:*:*:*:14.2:*:*"
      ]
    }
  ],
  "software": [
    {
      "id": 2363,
      "name": "Docker Desktop",
      "version": "4.9.1",
      "source": "programs",
      "browser": "",
      "generated_cpe": "cpe:2.3:a:docker:docker_desktop:4.9.1:*:*:*:*:windows:*:*",
      "hosts_count": 50,
      "resolved_in_version": "5.0.0"
    }
  ]
}
```


---

<meta name="description" value="Documentation for Fleet's vulnerabilities REST API endpoints.">
<meta name="pageOrderInSection" value="190">
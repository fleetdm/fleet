# Software

- [List all software](#list-all-software)
- [Count software](#count-software)

## List all software

`GET /api/v1/fleet/software`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                |
| ----------------------- | ------- | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                       |
| per_page                | integer | query | Results per page.                                                                                                                                                          |
| order_key               | string  | query | What to order results by. Allowed fields are `name`, `hosts_count`, `cve_published`, `cvss_score`, `epss_probability` and `cisa_known_exploit`. Default is `hosts_count` (descending).      |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                              |
| query                   | string  | query | Search query keywords. Searchable fields include `name`, `version`, and `cve`.                                                                                             |
| team_id                 | integer | query | _Available in Fleet Premium_ Filters the software to only include the software installed on the hosts that are assigned to the specified team.                             |
| vulnerable              | bool    | query | If true or 1, only list software that has detected vulnerabilities. Default is `false`.                                                                                    |

#### Example

`GET /api/v1/fleet/software`

##### Default response

`Status: 200`

```json
{
    "counts_updated_at": "2022-01-01 12:32:00",
    "software": [
      {
        "id": 1,
        "name": "glibc",
        "version": "2.12",
        "source": "rpm_packages",
        "release": "1.212.el6",
        "vendor": "CentOS",
        "arch": "x86_64",
        "generated_cpe": "cpe:2.3:a:gnu:glibc:2.12:*:*:*:*:*:*:*",
        "vulnerabilities": [
          {
            "cve": "CVE-2009-5155",
            "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2009-5155",
            "cvss_score": 7.5,
            "epss_probability": 0.01537,
            "cisa_known_exploit": false,
            "cve_published": "2022-01-01 12:32:00"
          }
        ],
        "hosts_count": 1
      }
    ]
}
```

## Count software

`GET /api/v1/fleet/software/count`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| query                   | string  | query | Search query keywords. Searchable fields include `name`, `version`, and `cve`.                                                                                                                                                                                                                                                               |
| team_id                 | integer | query | _Available in Fleet Premium_ Filters the software to only include the software installed on the hosts that are assigned to the specified team.                                                                                                                                                                                              |
| vulnerable              | bool    | query | If true or 1, only list software that has detected vulnerabilities.                                                                                                                                                                                                                                                                         |

#### Example

`GET /api/v1/fleet/software/count`

##### Default response

`Status: 200`

```json
{
  "count": 43
}
```

<meta name="description" value="Documentation for the software endpoint in Fleet's REST API.">
<meta name="pageOrderInSection" value="1300">
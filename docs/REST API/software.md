# Software

## List software

Get a list of all software.

`GET /api/v1/fleet/software/titles`

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                |
| ----------------------- | ------- | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                       |
| per_page                | integer | query | Results per page.                                                                                                                                                          |
| order_key               | string  | query | What to order results by. Allowed fields are `name` and `hosts_count`. Default is `hosts_count` (descending).                                                              |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                              |
| query                   | string  | query | Search query keywords. Searchable fields include `title` and `cve`.                                                                                                        |
| team_id                 | integer | query | _Available in Fleet Premium_. Filters the software to only include the software installed on the hosts that are assigned to the specified team. Use `0` to filter by hosts assigned to "No team".                            |
| vulnerable              | boolean | query | If true or 1, only list software that has detected vulnerabilities. Default is `false`.                                                                                    |
| available_for_install   | boolean | query | If `true` or `1`, only list software that is available for install (added by the user). Default is `false`.                                                                |
| self_service            | boolean | query | If `true` or `1`, only lists self-service software. Default is `false`.  |
| packages_only           | boolean | query | If `true` or `1`, only lists packages available for install (without App Store apps).  |
| min_cvss_score | integer | query | _Available in Fleet Premium_. Filters to include only software with vulnerabilities that have a CVSS version 3.x base score higher than the specified value.   |
| max_cvss_score | integer | query | _Available in Fleet Premium_. Filters to only include software with vulnerabilities that have a CVSS version 3.x base score lower than what's specified.   |
| exploit | boolean | query | _Available in Fleet Premium_. If `true`, filters to only include software with vulnerabilities that have been actively exploited in the wild (`cisa_known_exploit: true`). Default is `false`.  |

### Example

`GET /api/v1/fleet/software/titles?team_id=3`

#### Default response

`Status: 200`

```json
{
  "counts_updated_at": "2022-01-01 12:32:00",
  "count": 2,
  "software_titles": [
    {
      "id": 12,
      "name": "Firefox.app",
      "software_package": {
        "name": "FirefoxInsall.pkg",
        "version": "125.6",
        "self_service": true
      },
      "app_store_app": null,
      "versions_count": 3,
      "source": "apps",
      "browser": "",
      "hosts_count": 48,
      "versions": [
        {
          "id": 123,
          "version": "1.12",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
        },
        {
          "id": 124,
          "version": "3.4",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
        },
        {
          "id": 12
          "version": "1.13",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
        }
      ]
    },
    {
      "id": 22,
      "name": "Google Chrome.app",
      "software_package": null,
      "app_store_app": null,
      "versions_count": 5,
      "source": "apps",
      "browser": "",
      "hosts_count": 345,
      "versions": [
        {
          "id": 331,
          "version": "118.1",
          "vulnerabilities": ["CVE-2023-1234"]
        },
        {
          "id": 332,
          "version": "119.0",
          "vulnerabilities": ["CVE-2023-9876", "CVE-2023-2367"]
        },
        {
          "id": 334,
          "version": "119.4",
          "vulnerabilities": ["CVE-2023-1133", "CVE-2023-2224"]
        },
        {
          "id": 348,
          "version": "121.5",
          "vulnerabilities": ["CVE-2023-0987", "CVE-2023-5673", "CVE-2023-1334"]
        },
      ]
    },
    {
      "id": 32,
      "name": "1Password – Password Manager",
      "software_package": null,
      "app_store_app": null,
      "versions_count": 1,
      "source": "chrome_extensions",
      "browser": "chrome",
      "hosts_count": 345,
      "versions": [
        {
          "id": 4242,
          "version": "2.3.7",
          "vulnerabilities": []
        }
      ]
    }
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

## List software versions

Get a list of all software versions.

`GET /api/v1/fleet/software/versions`

### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                |
| ----------------------- | ------- | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                       |
| per_page                | integer | query | Results per page.                                                                                                                                                          |
| order_key               | string  | query | What to order results by. Allowed fields are `name`, `hosts_count`, `cve_published`, `cvss_score`, `epss_probability` and `cisa_known_exploit`. Default is `hosts_count` (descending).      |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                              |
| query                   | string  | query | Search query keywords. Searchable fields include `name`, `version`, and `cve`.                                                                                             |
| team_id                 | integer | query | _Available in Fleet Premium_. Filters the software to only include the software installed on the hosts that are assigned to the specified team. Use `0` to filter by hosts assigned to "No team".                             |
| vulnerable              | boolean    | query | If true or 1, only list software that has detected vulnerabilities. Default is `false`.                                                                                    |
| min_cvss_score | integer | query | _Available in Fleet Premium_. Filters to include only software with vulnerabilities that have a CVSS version 3.x base score higher than the specified value.   |
| max_cvss_score | integer | query | _Available in Fleet Premium_. Filters to only include software with vulnerabilities that have a CVSS version 3.x base score lower than what's specified.   |
| exploit | boolean | query | _Available in Fleet Premium_. If `true`, filters to only include software with vulnerabilities that have been actively exploited in the wild (`cisa_known_exploit: true`). Default is `false`.  |

### Example

`GET /api/v1/fleet/software/versions`

#### Default response

`Status: 200`

```json
{
    "counts_updated_at": "2022-01-01 12:32:00",
    "count": 1,
    "software": [
      {
        "id": 1,
        "name": "glibc",
        "version": "2.12",
        "source": "rpm_packages",
        "browser": "",
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
            "cve_published": "2022-01-01 12:32:00",
            "cve_description": "In the GNU C Library (aka glibc or libc6) before 2.28, parse_reg_exp in posix/regcomp.c misparses alternatives, which allows attackers to cause a denial of service (assertion failure and application exit) or trigger an incorrect result by attempting a regular-expression match.",
            "resolved_in_version": "2.28"
          }
        ],
        "hosts_count": 1
      },
      {
        "id": 2,
        "name": "1Password – Password Manager",
        "version": "2.10.0",
        "source": "chrome_extensions",
        "browser": "chrome",
        "extension_id": "aeblfdkhhhdcdjpifhhbdiojplfjncoa",
        "generated_cpe": "cpe:2.3:a:1password:1password:2.19.0:*:*:*:*:chrome:*:*",
        "hosts_count": 345,
        "vulnerabilities": null
      }
    ],
    "meta": {
      "has_next_results": false,
      "has_previous_results": false
    }
}
```

## List operating systems

Returns a list of all operating systems.

`GET /api/v1/fleet/os_versions`

### Parameters

| Name                | Type     | In    | Description                                                                                                                          |
| ---      | ---      | ---   | ---                                                                                                                                  |
| team_id             | integer | query | _Available in Fleet Premium_. Filters response data to the specified team. Use `0` to filter by hosts assigned to "No team".  |
| platform            | string   | query | Filters the hosts to the specified platform |
| os_name     | string | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                 |
| os_version    | string | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                 |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                       |
| per_page                | integer | query | Results per page.                                                                                                                                                          |
| order_key               | string  | query | What to order results by. Allowed fields are: `hosts_count`. Default is `hosts_count` (descending).      |
| order_direction | string | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |


#### Default response

`Status: 200`

```json
{
  "count": 1
  "counts_updated_at": "2023-12-06T22:17:30Z",
  "os_versions": [
    {
      "os_version_id": 123,
      "hosts_count": 21,
      "name": "Microsoft Windows 11 Pro 23H2 10.0.22621.1234",
      "name_only": "Microsoft Windows 11 Pro 23H2",
      "version": "10.0.22621.1234",
      "platform": "windows",
      "generated_cpes": [],
      "vulnerabilities": [
        {
          "cve": "CVE-2022-30190",
          "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2022-30190",
          "cvss_score": 7.8,// Available in Fleet Premium
          "epss_probability": 0.9729,// Available in Fleet Premium
          "cisa_known_exploit": false,// Available in Fleet Premium
          "cve_published": "2022-06-01T00:15:00Z",// Available in Fleet Premium
          "cve_description": "Microsoft Windows Support Diagnostic Tool (MSDT) Remote Code Execution Vulnerability.",// Available in Fleet Premium
          "resolved_in_version": ""// Available in Fleet Premium
        }
      ]
    }
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

OS vulnerability data is currently available for Windows and macOS. For other platforms, `vulnerabilities` will be an empty array:

```json
{
  "hosts_count": 1,
  "name": "CentOS Linux 7.9.2009",
  "name_only": "CentOS",
  "version": "7.9.2009",
  "platform": "rhel",
  "generated_cpes": [],
  "vulnerabilities": []
}
```

## Get software

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

Returns information about the specified software. By default, `versions` are sorted in descending order by the `hosts_count` field.

`GET /api/v1/fleet/software/titles/:id`

### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id   | integer | path | **Required.** The software title's ID. |
| team_id             | integer | query | _Available in Fleet Premium_. Filters response data to the specified team. Use `0` to filter by hosts assigned to "No team".  |

### Example

`GET /api/v1/fleet/software/titles/12`

#### Default response

`Status: 200`

```json
{
  "software_title": {
    "id": 12,
    "name": "Firefox.app",
    "bundle_identifier": "org.mozilla.firefox",
    "software_package": {
      "name": "FalconSensor-6.44.pkg",
      "version": "6.44",
      "installer_id": 23,
      "team_id": 3,
      "uploaded_at": "2024-04-01T14:22:58Z",
      "install_script": "sudo installer -pkg '$INSTALLER_PATH' -target /",
      "pre_install_query": "SELECT 1 FROM macos_profiles WHERE uuid='c9f4f0d5-8426-4eb8-b61b-27c543c9d3db';",
      "post_install_script": "sudo /Applications/Falcon.app/Contents/Resources/falconctl license 0123456789ABCDEFGHIJKLMNOPQRSTUV-WX",
      "uninstall_script": "/Library/CS/falconctl uninstall",
      "self_service": true,
      "status": {
        "installed": 3,
        "pending_install": 1,
        "failed_install": 0,
        "pending_uninstall": 2,
        "failed_uninstall": 1
      }
    },
    "app_store_app": null,
    "source": "apps",
    "browser": "",
    "hosts_count": 48,
    "versions": [
      {
        "id": 123,
        "version": "117.0",
        "vulnerabilities": ["CVE-2023-1234"],
        "hosts_count": 37
      },
      {
        "id": 124,
        "version": "116.0",
        "vulnerabilities": ["CVE-2023-4321"],
        "hosts_count": 7
      },
      {
        "id": 127,
        "version": "115.5",
        "vulnerabilities": ["CVE-2023-7654"],
        "hosts_count": 4
      }
    ]
  }
}
```

### Example (App Store app)

`GET /api/v1/fleet/software/titles/15`

#### Default response

`Status: 200`

```json
{
  "software_title": {
    "id": 15,
    "name": "Logic Pro",
    "bundle_identifier": "com.apple.logic10",
    "software_package": null,
    "app_store_app": {
      "name": "Logic Pro",
      "app_store_id": 1091189122,
      "latest_version": "2.04",
      "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
      "self_service": true,
      "status": {
        "installed": 3,
        "pending": 1,
        "failed": 2,
      }
    },
    "source": "apps",
    "browser": "",
    "hosts_count": 48,
    "versions": [
      {
        "id": 123,
        "version": "2.04",
        "vulnerabilities": [],
        "hosts_count": 24
      }
    ]
  }
}
```

## Get software version

Returns information about the specified software version.

`GET /api/v1/fleet/software/versions/:id`

### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id   | integer | path | **Required.** The software version's ID. |
| team_id             | integer | query | _Available in Fleet Premium_. Filters response data to the specified team. Use `0` to filter by hosts assigned to "No team".  |

### Example

`GET /api/v1/fleet/software/versions/12`

#### Default response

`Status: 200`

```json
{
  "software": {
    "id": 425224,
    "name": "Firefox.app",
    "version": "117.0",
    "bundle_identifier": "org.mozilla.firefox",
    "source": "apps",
    "browser": "",
    "generated_cpe": "cpe:2.3:a:mozilla:firefox:117.0:*:*:*:*:macos:*:*",
    "vulnerabilities": [
      {
        "cve": "CVE-2023-4863",
        "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2023-4863",
        "created_at": "2024-07-01T00:15:00Z",
        "cvss_score": 8.8, // Available in Fleet Premium
        "epss_probability": 0.4101, // Available in Fleet Premium
        "cisa_known_exploit": true, // Available in Fleet Premium
        "cve_published": "2023-09-12T15:15:00Z", // Available in Fleet Premium
        "resolved_in_version": "" // Available in Fleet Premium
      },
      {
        "cve": "CVE-2023-5169",
        "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2023-5169",
        "created_at": "2024-07-01T00:15:00Z",
        "cvss_score": 6.5, // Available in Fleet Premium
        "epss_probability": 0.00073, // Available in Fleet Premium
        "cisa_known_exploit": false, // Available in Fleet Premium
        "cve_published": "2023-09-27T15:19:00Z", // Available in Fleet Premium
        "resolved_in_version": "118" // Available in Fleet Premium
      }
    ]
  }
}
```


## Get operating system version

Retrieves information about the specified operating system (OS) version.

`GET /api/v1/fleet/os_versions/:id`

### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id   | integer | path | **Required.** The OS version's ID. |
| team_id             | integer | query | _Available in Fleet Premium_. Filters response data to the specified team. Use `0` to filter by hosts assigned to "No team".  |

#### Default response

`Status: 200`

```json
{
  "counts_updated_at": "2023-12-06T22:17:30Z",
  "os_version": {
    "id": 123,
    "hosts_count": 21,
    "name": "Microsoft Windows 11 Pro 23H2 10.0.22621.1234",
    "name_only": "Microsoft Windows 11 Pro 23H2",
    "version": "10.0.22621.1234",
    "platform": "windows",
    "generated_cpes": [],
    "vulnerabilities": [
      {
        "cve": "CVE-2022-30190",
        "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2022-30190",
        "created_at": "2024-07-01T00:15:00Z",
        "cvss_score": 7.8,// Available in Fleet Premium
        "epss_probability": 0.9729,// Available in Fleet Premium
        "cisa_known_exploit": false,// Available in Fleet Premium
        "cve_published": "2022-06-01T00:15:00Z",// Available in Fleet Premium
        "cve_description": "Microsoft Windows Support Diagnostic Tool (MSDT) Remote Code Execution Vulnerability.",// Available in Fleet Premium
        "resolved_in_version": ""// Available in Fleet Premium
      }
    ]
  }
}
```

OS vulnerability data is currently available for Windows and macOS. For other platforms, `vulnerabilities` will be an empty array:

```json
{
  "id": 321,
  "hosts_count": 1,
  "name": "CentOS Linux 7.9.2009",
  "name_only": "CentOS",
  "version": "7.9.2009",
  "platform": "rhel",
  "generated_cpes": [],
  "vulnerabilities": []
}
```

## Add package

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

Add a package (.pkg, .msi, .exe, .deb) to install on macOS, Windows, or Linux (Ubuntu) hosts.


`POST /api/v1/fleet/software/package`

### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| software        | file    | form | **Required**. Installer package file. Supported packages are PKG, MSI, EXE, and DEB.   |
| team_id         | integer | form | **Required**. The team ID. Adds a software package to the specified team. |
| install_script  | string | form | Script that Fleet runs to install software. If not specified Fleet runs [default install script](https://github.com/fleetdm/fleet/tree/f71a1f183cc6736205510580c8366153ea083a8d/pkg/file/scripts) for each package type. |
| pre_install_query  | string | form | Query that is pre-install condition. If the query doesn't return any result, Fleet won't proceed to install. |
| post_install_script | string | form | The contents of the script to run after install. If the specified script fails (exit code non-zero) software install will be marked as failed and rolled back. |
| self_service | boolean | form | Self-service software is optional and can be installed by the end user. |

### Example

`POST /api/v1/fleet/software/package`

#### Request header

```http
Content-Length: 8500
Content-Type: multipart/form-data; boundary=------------------------d8c247122f594ba0
```

#### Request body

```http
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="team_id"
1
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="self_service"
true
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="install_script"
sudo installer -pkg /temp/FalconSensor-6.44.pkg -target /
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="pre_install_query"
SELECT 1 FROM macos_profiles WHERE uuid='c9f4f0d5-8426-4eb8-b61b-27c543c9d3db';
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="post_install_script"
sudo /Applications/Falcon.app/Contents/Resources/falconctl license 0123456789ABCDEFGHIJKLMNOPQRSTUV-WX
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="software"; filename="FalconSensor-6.44.pkg"
Content-Type: application/octet-stream
<BINARY_DATA>
--------------------------d8c247122f594ba0
```

#### Default response

`Status: 200`

## Modify package

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

Update a package to install on macOS, Windows, or Linux (Ubuntu) hosts.

`PATCH /api/v1/fleet/software/titles/:title_id/package`

### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| software        | file    | form | Installer package file. Supported packages are PKG, MSI, EXE, and DEB.   |
| team_id         | integer | form | **Required**. The team ID. Updates a software package in the specified team. |
| install_script  | string | form | Command that Fleet runs to install software. If not specified Fleet runs the [default install command](https://github.com/fleetdm/fleet/tree/f71a1f183cc6736205510580c8366153ea083a8d/pkg/file/scripts) for each package type. |
| pre_install_query  | string | form | Query that is pre-install condition. If the query doesn't return any result, the package will not be installed. |
| post_install_script | string | form | The contents of the script to run after install. If the specified script fails (exit code non-zero) software install will be marked as failed and rolled back. |
| self_service | boolean | form | Whether this is optional self-service software that can be installed by the end user. |

> Changes to the installer package will reset installation counts. Changes to any field other than `self_service` will cancel pending installs for the old package.
### Example

`PATCH /api/v1/fleet/software/titles/1/package`

#### Request header

```http
Content-Length: 8500
Content-Type: multipart/form-data; boundary=------------------------d8c247122f594ba0
```

#### Request body

```http
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="team_id"
1
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="self_service"
true
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="install_script"
sudo installer -pkg /temp/FalconSensor-6.44.pkg -target /
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="pre_install_query"
SELECT 1 FROM macos_profiles WHERE uuid='c9f4f0d5-8426-4eb8-b61b-27c543c9d3db';
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="post_install_script"
sudo /Applications/Falcon.app/Contents/Resources/falconctl license 0123456789ABCDEFGHIJKLMNOPQRSTUV-WX
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="software"; filename="FalconSensor-6.44.pkg"
Content-Type: application/octet-stream
<BINARY_DATA>
--------------------------d8c247122f594ba0
```

#### Default response

`Status: 200`

```json
{
  "software_package": {
    "name": "FalconSensor-6.44.pkg",
    "version": "6.44",
    "installer_id": 23,
    "team_id": 3,
    "uploaded_at": "2024-04-01T14:22:58Z",
    "install_script": "sudo installer -pkg /temp/FalconSensor-6.44.pkg -target /",
    "pre_install_query": "SELECT 1 FROM macos_profiles WHERE uuid='c9f4f0d5-8426-4eb8-b61b-27c543c9d3db';",
    "post_install_script": "sudo /Applications/Falcon.app/Contents/Resources/falconctl license 0123456789ABCDEFGHIJKLMNOPQRSTUV-WX",
    "self_service": true,
    "status": {
      "installed": 0,
      "pending": 0,
      "failed": 0
    }
  }
}
```

## List App Store apps

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

Returns the list of Apple App Store (VPP) that can be added to the specified team. If an app is already added to the team, it's excluded from the list.

`GET /api/v1/fleet/software/app_store_apps`

### Parameters

| Name    | Type | In | Description |
| ------- | ---- | -- | ----------- |
| team_id | integer | query | **Required**. The team ID. |

### Example

`GET /api/v1/fleet/software/app_store_apps/?team_id=3`

#### Default response

`Status: 200`

```json
{
  "app_store_apps": [
    {
      "name": "Xcode",
      "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
      "latest_version": "15.4",
      "app_store_id": "497799835",
      "platform": "darwin"
    },
    {
      "name": "Logic Pro",
      "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
      "latest_version": "2.04",
      "app_store_id": "634148309",
      "platform": "ios"
    },
    {
      "name": "Logic Pro",
      "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
      "latest_version": "2.04",
      "app_store_id": "634148309",
      "platform": "ipados"
    },
  ]
}
```

## Add App Store app

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

Add App Store (VPP) app purchased in Apple Business Manager.

`POST /api/v1/fleet/software/app_store_apps`

### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| app_store_id   | string | body | **Required.** The ID of App Store app. |
| team_id       | integer | body | **Required**. The team ID. Adds VPP software to the specified team.  |
| platform | string | body | The platform of the app (`darwin`, `ios`, or `ipados`). Default is `darwin`. |
| self_service | boolean | body | Self-service software is optional and can be installed by the end user. |

### Example

`POST /api/v1/fleet/software/app_store_apps?team_id=3`

#### Request body

```json
{
  "app_store_id": "497799835",
  "team_id": 2,
  "platform": "ipados"
  "self_service": true
}
```

#### Default response

`Status: 200`

## Install package or App Store app

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

Install software (package or App Store app) on a macOS, iOS, iPadOS, Windows, or Linux (Ubuntu) host. Software title must have a `software_package` or `app_store_app` added to be installed.

`POST /api/v1/fleet/hosts/:id/software/:software_title_id/install`

### Parameters

| Name              | Type       | In   | Description                                      |
| ---------         | ---------- | ---- | --------------------------------------------     |
| id                | integer    | path | **Required**. The host's ID.                     |
| software_title_id | integer    | path | **Required**. The software title's ID.           |

### Example

`POST /api/v1/fleet/hosts/123/software/3435/install`

#### Default response

`Status: 202`

## Uninstall package

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.
_Available in Fleet Premium._

Uninstall software (package) on a macOS, Windows, or Linux (Ubuntu) host. Software title must have a `software_package` added to be uninstalled.

`POST /api/v1/fleet/hosts/:id/software/:software_title_id/uninstall`

### Parameters

| Name              | Type       | In   | Description                                      |
| ---------         | ---------- | ---- | --------------------------------------------     |
| id                | integer    | path | **Required**. The host's ID.                     |
| software_title_id | integer    | path | **Required**. The software title's ID.           |

### Example

`POST /api/v1/fleet/hosts/123/software/3435/uninstall`

#### Default response

`Status: 202`

## Get package install result

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

`GET /api/v1/fleet/software/install/:install_uuid/results`

Get the results of a software package install.

To get the results of an App Store app install, use the [List MDM commands](#list-mdm-commands) and [Get MDM command results](#get-mdm-command-results) API enpoints. Fleet uses an MDM command to install App Store apps.

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| install_uuid | string | path | **Required**. The software installation UUID.|

### Example

`GET /api/v1/fleet/software/install/b15ce221-e22e-4c6a-afe7-5b3400a017da/results`

#### Default response

`Status: 200`

```json
 {
   "install_uuid": "b15ce221-e22e-4c6a-afe7-5b3400a017da",
   "software_title": "Falcon.app",
   "software_title_id": 8353,
   "software_package": "FalconSensor-6.44.pkg",
   "host_id": 123,
   "host_display_name": "Marko's MacBook Pro",
   "status": "failed",
   "output": "Installing software...\nError: The operation can’t be completed because the item “Falcon” is in use.",
   "pre_install_query_output": "Query returned result\nSuccess",
   "post_install_script_output": "Running script...\nExit code: 1 (Failed)\nRolling back software install...\nSuccess"
 }
```

## Download package

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

`GET /api/v1/fleet/software/titles/:software_title_id/package?alt=media`

### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| software_title_id   | integer | path | **Required**. The ID of the software title to download software package.|
| team_id | integer | query | **Required**. The team ID. Downloads a software package added to the specified team. |
| alt             | integer | query | **Required**. If specified and set to "media", downloads the specified software package. |

### Example

`GET /api/v1/fleet/software/titles/123/package?alt=media?team_id=2`

#### Default response

`Status: 200`

```http
Status: 200
Content-Type: application/octet-stream
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

## Delete package or App Store app

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

Deletes software that's available for install (package or App Store app).

`DELETE /api/v1/fleet/software/titles/:software_title_id/available_for_install`

### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| software_title_id              | integer | path | **Required**. The ID of the software title to delete software available for install. |
| team_id | integer | query | **Required**. The team ID. Deletes a software package added to the specified team. |

### Example

`DELETE /api/v1/fleet/software/titles/24/available_for_install?team_id=2`

#### Default response

`Status: 204`

---

<meta name="description" value="Documentation for Fleet's software REST API endpoints.">
<meta name="pageOrderInSection" value="150">
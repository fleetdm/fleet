# Outbound requests from Fleet

This guide lists all external URLs and API endpoints that the Fleet server makes outbound requests to. This is useful for configuring firewall rules, allowlists, and understanding Fleet's network dependencies.

Find out which Fleet endpoints to [expose to the public internet](https://fleetdm.com/guides/what-api-endpoints-to-expose-to-the-public-internet).

## Fleet-hosted proxies

Fleet routes some external API calls through proxies hosted on `fleetdm.com`. These proxies manage authentication when only Fleet can access these external services.

### Apple VPP app metadata

- `https://fleetdm.com/api/vpp/v1/metadata/{region}`: Fetches app metadata for App Store (VPP) apps from Apple.
- `https://fleetdm.com/api/vpp/v1/auth`: Authenticates with the proxy.

Upstream API endpoint: `https://api.ent.apple.com/v1/catalog/{region}/stoken-authenticated-apps`

To bypass the proxy and request from Apple directly, set the [mdm.apple_vpp_app_metadata_api_bearer_token](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-apple-vpp-app-metadata-api-bearer-token) server config option.

### Android MDM

- `https://fleetdm.com/api/android/`: Proxies requests to Google's Android Management API.

Upstream API: `https://androidmanagement.googleapis.com/v1/`

### Microsoft compliance partner

- `https://fleetdm.com/api/v1/microsoft-compliance-partner`: Proxies device compliance status updates to Microsoft Entra for conditional access enforcement.

To use a different proxy, set the `microsoft_compliance_partner.proxy_uri` and `microsoft_compliance_partner.proxy_api_key` server config options.

### Osquery policy autofill

- `https://fleetdm.com/api/v1/get-human-interpretation-from-osquery-sql`: AI-powered human interpretation of osquery SQL for policies. This is Fleet-specific service. This is optional. Only used if enabled.

## Apple MDM

### Apple DEP / Apple Business Manager (ABM)

All ABM/DEP endpoints use the base URL `https://mdmenrollment.apple.com/` and include:

- `/session`: Authenticate and get session token.
- `/account`: Get organization/account details.
- `/server/devices`: Fetch all assigned devices.
- `/devices/sync`: Sync device updates.
- `/devices`: Get specific device details.
- `/profile/devices`: Assign profile to devices.
- `/profile`: Define/create a profile.
- `/profile?profile_uuid={uuid}`: Get profile details.
- `/account-driven-enrollment/profile`: Fetch or assign Account Driven Enrollment (ADE) service discovery.

### Apple Push Notification Service (APNs)

- `https://api.push.apple.com` (production)
- `https://api.development.push.apple.com` (development)

Both endpoints are also available on port `2197`.

### Apple software updates (GDMF)

- `https://gdmf.apple.com/v2/pmv`: Fetches Apple OS update metadata and supported OS versions.

### Apple Volume Purchase Program (VPP)

- `https://vpp.itunes.apple.com/mdm/v2`: Manages App Store app licenses and device assignments.

## Google services

### Google Calendar

- `https://www.googleapis.com/calendar/v3/`: Calendar API v3 for managing scheduled maintenance events.
- `https://oauth2.googleapis.com/token`: OAuth2 JWT token exchange.

### Google Cloud Storage

- `https://storage.googleapis.com`: File storage via S3-compatible API. Only used if GCS is configured as the file storage backend.

## Microsoft services

### MSRC (Microsoft Security Response Center)

- `https://api.msrc.microsoft.com`: Fetches Microsoft security bulletins and vulnerability data.

## Vulnerability data sources

### NVD (National Vulnerability Database)

- `https://services.nvd.nist.gov/rest/json/cves/2.0`: Fetches CVE vulnerability data from NIST.

### VulnCheck

- `https://api.vulncheck.com/v3/backup/nist-nvd2`: Backup source for vulnerability data.

### CISA Known Exploited Vulnerabilities

- `https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json`: Known exploited vulnerabilities catalog.

### Fleet vulnerability data (GitHub)

- `https://github.com/fleetdm/nvd/releases`: Downloads CVE feeds, CPE databases, CPE translations, MSRC bulletins, MacOffice release notes, and OVAL source mappings.
- `https://github.com/fleetdm/vulnerabilities/releases`: Downloads pre-processed CVE data.

### OVAL definitions by Linux distribution

**Ubuntu:**
- `https://security-metadata.canonical.com/oval/oci.com.ubuntu.{version}.cve.oval.xml.bz2`

**Debian:**
- `https://www.debian.org/security/oval/oval-definitions-{version}.xml.bz2`

**SUSE:**
- `https://ftp.suse.com/pub/projects/security/oval/{product}.xml.gz`

**Oracle Linux:**
- `https://linux.oracle.com/security/oval/com.oracle.elsa-all.xml.bz2`

**Alpine:**
- `https://secdb.alpinelinux.org/v{version}/main.yaml`
- `https://secdb.alpinelinux.org/v{version}/community.yaml`

**Fedora:**
- `https://dl.fedoraproject.org/pub/fedora/linux/updates/{version}/Everything/{arch}/repodata/repomd.xml`
- `https://archives.fedoraproject.org/pub/archive/fedora/linux/updates/{version}/Everything/{arch}/repodata/repomd.xml`

**Amazon Linux 1:**
- `http://repo.us-west-2.amazonaws.com/2018.03/updates/x86_64/mirror.list`

**Amazon Linux 2:**
- `https://cdn.amazonlinux.com/2/core/latest/x86_64/mirror.list`

**Amazon Linux 2022:**
- `https://cdn.amazonlinux.com/al2022/core/mirrors/latest/x86_64/mirror.list`

**Amazon Linux 2023:**
- `https://cdn.amazonlinux.com/al2023/core/mirrors/latest/x86_64/mirror.list`

## Issue tracking integrations (user-configured)

### Jira

- `https://{instance}.atlassian.net`: Creates Jira issues for failing policies and vulnerabilities. The instance URL is configured by the Fleet admin.

### Zendesk

- `https://{subdomain}.zendesk.com`: Creates Zendesk tickets for failing policies and vulnerabilities. The subdomain is configured by the Fleet admin.

## Certificate authorities

### DigiCert

- `https://one.digicert.com/mpki/api/v2/`: Generates and manages certificates via DigiCert MPKI. The URL is user-configured.

## AWS services (user-configured)

All AWS endpoints are regional and configured by the Fleet admin.

- **S3**: Stores software installers, bootstrap packages, software icons, and carves. Supports custom endpoints for S3-compatible services (MinIO, etc.).
- **SES**: Sends email notifications.
- **Kinesis**: Streams host activity logs.
- **Firehose**: Delivers host activity logs.

## Other user-configured endpoints

- **Custom webhook URLs (HTTPS)**: Activity webhooks, failing policy webhooks, and vulnerability webhooks.
- **SMTP servers**: Sends email invitations, password resets, and notifications.
- **Kafka REST proxy**: Streams logs to Kafka.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="authorFullName" value="Marko Lisica">
<meta name="publishedOn" value="2026-04-03">
<meta name="articleTitle" value="Outbound requests from Fleet">
<meta name="description" value="List of all external URLs and API endpoints that the Fleet server makes outbound requests to.">

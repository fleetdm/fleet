// Allowlist of pilot endpoints to include in the generated OpenAPI 3.1 spec.
//
// The PoC ships ONE entry. Adding the remaining pilot endpoints from
// https://github.com/fleetdm/fleet/issues/45279 is a data-only change: append
// to this array. The Markdown parser keys off `markdownHeading` and extracts:
//   - the request line (`HTTP_METHOD /api/v1/...`)
//   - the `#### Parameters` table directly under the heading
//   - the first ```json``` block under `##### Default response`
//
// Each entry may also override or supplement what the parser detects. Hand-
// authored overrides are explicitly marked so a future contributor knows what
// to delete if/when parser support catches up to the Markdown reality.
'use strict';

/** @typedef {{
 *   markdownHeading: string,          // exact "### List hosts" style line, no trailing punctuation
 *   operationId: string,              // OpenAPI operationId — must be unique
 *   tag: string,                      // OpenAPI tag (groups related endpoints)
 *   summary: string,                  // one-line summary (used as the operation summary)
 *   description?: string,             // optional longer description
 *   pathOverride?: string,            // when the doc's request line uses :id, force /{id} form
 *   pathParameters?: Array<{ name: string, type: string, description: string }>,
 * }} EndpointSpec */

/** @type {EndpointSpec[]} */
const endpoints = [
  {
    markdownHeading: '### List hosts',
    operationId: 'listHosts',
    tag: 'Hosts',
    summary: 'List hosts',
    description:
      'Returns a paginated list of hosts enrolled in Fleet. Supports filtering, ' +
      'ordering, and optional inclusion of related data (software, policies, ' +
      'users, labels, device status).',
  },
  {
    markdownHeading: '### Count hosts',
    operationId: 'countHosts',
    tag: 'Hosts',
    summary: 'Count hosts',
  },
  {
    markdownHeading: '### Get hosts summary',
    operationId: 'getHostsSummary',
    tag: 'Hosts',
    summary: 'Get hosts summary',
    description: 'Returns the count of all hosts organized by status.',
  },
  {
    markdownHeading: '### Get host',
    operationId: 'getHost',
    tag: 'Hosts',
    summary: 'Get host',
    description: 'Returns the information of the specified host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: '### Get host by identifier',
    operationId: 'getHostByIdentifier',
    tag: 'Hosts',
    summary: 'Get host by identifier',
    description:
      "Returns the information of the host specified using the hostname, uuid, or hardware_serial as an identifier.",
    pathParameters: [
      { name: 'identifier', type: 'string', description: "The host's hostname, uuid, or hardware_serial." },
    ],
  },
  {
    markdownHeading: '### Get host by Fleet Desktop token',
    operationId: 'getHostByFleetDesktopToken',
    tag: 'Hosts',
    summary: 'Get host by Fleet Desktop token',
    description: 'Returns a subset of information about the host specified by token.',
    pathParameters: [
      { name: 'token', type: 'string', description: "The host's Fleet Desktop token." },
    ],
  },
  {
    markdownHeading: '### Delete host',
    operationId: 'deleteHost',
    tag: 'Hosts',
    summary: 'Delete host',
    description: 'Deletes the specified host from Fleet.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: '### Refetch host',
    operationId: 'refetchHost',
    tag: 'Hosts',
    summary: 'Refetch host',
    description: 'Flags the host details, labels and policies to be refetched the next time the host checks in.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: '### Refetch host by Fleet Desktop token',
    operationId: 'refetchHostByFleetDesktopToken',
    tag: 'Hosts',
    summary: 'Refetch host by Fleet Desktop token',
    pathParameters: [
      { name: 'token', type: 'string', description: "The host's Fleet Desktop token." },
    ],
  },
  {
    markdownHeading: "### Update hosts' fleet",
    operationId: 'transferHosts',
    tag: 'Hosts',
    summary: "Update hosts' fleet",
    description: 'Transfers hosts to a specified fleet.',
  },
  {
    markdownHeading: "### Update hosts' fleet by filter",
    operationId: 'transferHostsByFilter',
    tag: 'Hosts',
    summary: "Update hosts' fleet by filter",
    description: 'Transfers hosts matching the given filter to a specified fleet.',
  },
  {
    markdownHeading: "### Turn off host's MDM",
    operationId: 'turnOffHostMDM',
    tag: 'Hosts',
    summary: "Turn off host's MDM",
    description: 'Turns off MDM for the specified host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's ID in Fleet." },
    ],
  },
  {
    markdownHeading: '### Batch-delete hosts',
    operationId: 'batchDeleteHosts',
    tag: 'Hosts',
    summary: 'Batch-delete hosts',
    description: 'Delete hosts selected by filter or ids.',
  },
  {
    markdownHeading: '### Update human-device mapping',
    operationId: 'updateHumanDeviceMapping',
    tag: 'Hosts',
    summary: 'Update human-device mapping',
    description: 'Updates the email for the data source in the human-device mapping.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: "### Get host's device health report",
    operationId: 'getHostDeviceHealthReport',
    tag: 'Hosts',
    summary: "Get host's device health report",
    description: "Retrieves information about a single host's device health.",
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: "### Get host's mobile device management (MDM) information",
    operationId: 'getHostMDMInfo',
    tag: 'Hosts',
    summary: "Get host's MDM information",
    description: "Retrieves a host's MDM enrollment status and MDM server URL.",
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: '### Get mobile device management (MDM) status',
    operationId: 'getMDMStatus',
    tag: 'Hosts',
    summary: 'Get MDM status',
    description: 'Retrieves MDM enrollment summary.',
  },
  {
    markdownHeading: "### Get host's mobile device management (MDM) and Munki information",
    operationId: 'getHostMacadmins',
    tag: 'Hosts',
    summary: "Get host's MDM and Munki information",
    description: "Retrieves a host's MDM enrollment status, MDM server URL, and Munki version.",
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: "### Get hosts' aggregate mobile device management (MDM) and Munki information",
    operationId: 'getAggregateMacadmins',
    tag: 'Hosts',
    summary: "Get aggregate MDM and Munki information",
    description: 'Retrieves MDM enrollment status and Munki versions, aggregated across all hosts.',
  },
  {
    markdownHeading: "### Get host's software",
    operationId: 'getHostSoftware',
    tag: 'Hosts',
    summary: "Get host's software",
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's ID." },
    ],
  },
  {
    markdownHeading: '### Get hosts report in CSV',
    operationId: 'getHostsReportCSV',
    tag: 'Hosts',
    summary: 'Get hosts report in CSV',
    description: 'Returns the list of hosts in CSV format.',
  },
  {
    markdownHeading: "### Get host's disk encryption key",
    operationId: 'getHostDiskEncryptionKey',
    tag: 'Hosts',
    summary: "Get host's disk encryption key",
    description: 'Retrieves the disk encryption key for a host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: "### Get host's Recovery Lock password",
    operationId: 'getHostRecoveryLockPassword',
    tag: 'Hosts',
    summary: "Get host's Recovery Lock password",
    description: 'Retrieves the Recovery Lock password for a host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: "### Rotate host's Recovery Lock password",
    operationId: 'rotateHostRecoveryLockPassword',
    tag: 'Hosts',
    summary: "Rotate host's Recovery Lock password",
    description: 'Rotates the Recovery Lock password for a host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: "### Get host's certificates",
    operationId: 'getHostCertificates',
    tag: 'Hosts',
    summary: "Get host's certificates",
    description: 'Retrieves the certificates installed on a host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: "### Get host's OS settings (configuration profile)",
    operationId: 'getHostOSSettings',
    tag: 'Hosts',
    summary: "Get host's OS settings",
    description: 'Retrieves a list of the configuration profiles assigned to a host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's ID." },
    ],
  },
  {
    markdownHeading: '### Lock host',
    operationId: 'lockHost',
    tag: 'Hosts',
    summary: 'Lock host',
    description: 'Sends a command to lock the specified host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: 'ID of the host to be locked.' },
    ],
  },
  {
    markdownHeading: '### Unlock host',
    operationId: 'unlockHost',
    tag: 'Hosts',
    summary: 'Unlock host',
    description: 'Sends a command to unlock the specified host, or retrieves the unlock PIN for a macOS host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: 'ID of the host to be unlocked.' },
    ],
  },
  {
    markdownHeading: '### Wipe host',
    operationId: 'wipeHost',
    tag: 'Hosts',
    summary: 'Wipe host',
    description: 'Sends a command to wipe the specified host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: 'ID of the host to be wiped.' },
    ],
  },
  {
    markdownHeading: "### Get host's past activity",
    operationId: 'getHostPastActivity',
    tag: 'Hosts',
    summary: "Get host's past activity",
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's ID." },
    ],
  },
  {
    markdownHeading: "### Get host's upcoming activity",
    operationId: 'getHostUpcomingActivity',
    tag: 'Hosts',
    summary: "Get host's upcoming activity",
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: "### Cancel host's upcoming activity",
    operationId: 'cancelHostUpcomingActivity',
    tag: 'Hosts',
    summary: "Cancel host's upcoming activity",
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's ID." },
      { name: 'activity_id', type: 'string', description: "The ID of the host's upcoming activity." },
    ],
  },
  {
    markdownHeading: '### Add labels to host',
    operationId: 'addLabelsToHost',
    tag: 'Hosts',
    summary: 'Add labels to host',
    description: 'Adds manual labels to a host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: '### Remove labels from host',
    operationId: 'removeLabelsFromHost',
    tag: 'Hosts',
    summary: 'Remove labels from host',
    description: 'Removes manual labels from a host.',
    pathParameters: [
      { name: 'id', type: 'integer', description: "The host's id." },
    ],
  },
  {
    markdownHeading: '### Run live query on host (ad hoc)',
    operationId: 'runLiveQueryOnHost',
    tag: 'Hosts',
    summary: 'Run live query on host (ad hoc)',
    description: 'Runs an ad hoc live query against the specified host and responds with the results.',
    pathParameters: [
      { name: 'id', type: 'integer', description: 'The target host ID.' },
    ],
  },
  {
    markdownHeading: '### Run live query on host by identifier (ad hoc)',
    operationId: 'runLiveQueryOnHostByIdentifier',
    tag: 'Hosts',
    summary: 'Run live query on host by identifier (ad hoc)',
    description: 'Runs an ad hoc live query against a host identified by uuid and responds with the results.',
    pathParameters: [
      { name: 'identifier', type: 'string', description: "The host's hardware_serial, uuid, osquery_host_id, hostname, or node_key." },
    ],
  },
  {
    markdownHeading: "### Bypass host's conditional access",
    operationId: 'bypassHostConditionalAccess',
    tag: 'Hosts',
    summary: "Bypass host's conditional access",
    description: 'Grant a blocked host access for a single login.',
    pathParameters: [
      { name: 'token', type: 'string', description: "The host's device authentication token." },
    ],
  },
];

module.exports = { endpoints };

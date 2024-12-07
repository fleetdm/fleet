module.exports = {


  friendlyName: 'Receive usage analytics',


  description: 'Receive anonymous usage analytics from deployments of Fleet running in production.  (Not fleetctl preview or dev-mode deployments.)',


  inputs: {
    anonymousIdentifier: { required: true, type: 'string', example: '9pnzNmrES3mQG66UQtd29cYTiX2+fZ4CYxDvh495720=', description: 'An anonymous identifier telling us which Fleet deployment this is.', },
    fleetVersion: { required: true, type: 'string', example: 'x.x.x' },
    licenseTier: { type: 'string', isIn: ['free', 'premium', 'unknown'], defaultsTo: 'unknown' },
    numHostsEnrolled: { required: true, type: 'number', min: 0, custom: (num) => Math.floor(num) === num },
    numUsers: { type: 'number', defaultsTo: 0 },
    numTeams: { type: 'number', defaultsTo: 0 },
    numPolicies: { type: 'number', defaultsTo: 0 },
    numLabels: { type: 'number', defaultsTo: 0 },
    softwareInventoryEnabled: { type: 'boolean', defaultsTo: false },
    vulnDetectionEnabled: { type: 'boolean', defaultsTo: false },
    systemUsersEnabled: { type: 'boolean', defaultsTo: false },
    hostsStatusWebHookEnabled: { type: 'boolean', defaultsTo: false },
    numWeeklyActiveUsers: { type: 'number', defaultsTo: 0 },
    numWeeklyPolicyViolationDaysActual: { type: 'number', defaultsTo: 0 },
    numWeeklyPolicyViolationDaysPossible: { type: 'number', defaultsTo: 0 },
    hostsEnrolledByOperatingSystem: { type: {}, defaultsTo: {} },
    hostsEnrolledByOrbitVersion: { type: [{orbitVersion: 'string', numHosts: 'number'}], defaultsTo: [] }, // TODO: The name of this parameter does not match naming conventions.
    hostsEnrolledByOsqueryVersion: { type: [{osqueryVersion: 'string', numHosts: 'number'}], defaultsTo: [] }, // TODO: The name of this parameter does not match naming conventions.
    storedErrors: { type: [{}], defaultsTo: [] }, // TODO migrate all rows that have "[]" to {}
    numHostsNotResponding: { type: 'number', defaultsTo: 0, description: 'The number of hosts per deployment that have not submitted results for distibuted queries. A host is counted as not responding if Fleet hasn\'t received a distributed write to requested distibuted queries for the host during the 2-hour interval since the host was last seen. Hosts that have not been seen for 7 days or more are not counted.', },
    organization: { type: 'string', defaultsTo: 'unknown', description: 'For Fleet Premium deployments, the organization registered with the license.', },
    mdmMacOsEnabled: {type: 'boolean', defaultsTo: false},
    mdmWindowsEnabled: {type: 'boolean', defaultsTo: false},
    liveQueryDisabled: {type: 'boolean', defaultsTo: false},
    hostExpiryEnabled: {type: 'boolean', defaultsTo: false},
    numSoftwareVersions: {type: 'number', defaultsTo: 0},
    numHostSoftwares: {type: 'number', defaultsTo: 0},
    numSoftwareTitles: {type: 'number', defaultsTo: 0},
    numHostSoftwareInstalledPaths: {type: 'number', defaultsTo: 0},
    numSoftwareCPEs: {type: 'number', defaultsTo: 0},
    numSoftwareCVEs: {type: 'number', defaultsTo: 0},
    aiFeaturesDisabled: {type: 'boolean', defaultsTo: false },
    maintenanceWindowsEnabled: {type: 'boolean', defaultsTo: false },
    maintenanceWindowsConfigured: {type: 'boolean', defaultsTo: false },
    numHostsFleetDesktopEnabled: {type: 'number', defaultsTo: 0 },
    numQueries: {type: 'number', defaultsTo: 0 },
  },


  exits: {
    success: { description: 'Analytics data was stored successfully.' },
  },


  fn: async function (inputs) {
    // If organization was reported as an empty string, set it to the default value.
    if(inputs.organization === '') {
      inputs.organization = 'unknown';
    }
    // Create a database record for these usage statistics.
    await HistoricalUsageSnapshot.create(Object.assign({}, inputs));

  }


};

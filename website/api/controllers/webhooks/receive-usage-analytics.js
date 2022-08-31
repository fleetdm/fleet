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
    hostStatusWebhookEnabled: { type: 'boolean', defaultsTo: false },
    numWeeklyActiveUsers: { type: 'number', defaultsTo: 0 },
    hostsEnrolledByOperatingSystem: { type: 'json', defaultsTo: {} },
    storedErrors: { type: 'json', defaultsTo: '[]' },
    numHostsNotResponding: { type: 'number', defaultsTo: 0, description: 'The number of hosts per deployment that have not submitted results for distibuted queries. A host is counted as not responding if Fleet hasn\'t received a distributed write to requested distibuted queries for the host during the 2-hour interval since the host was last seen. Hosts that have not been seen for 7 days or more are not counted.', },
    organization: { type: 'string', defaultsTo: 'unknown', description: 'For Fleet Premium deployments, the organization registered with the license.', },
  },


  exits: {
    success: { description: 'Analytics data was stored successfully.' },
  },


  fn: async function (inputs) {

    await HistoricalUsageSnapshot.create(inputs);

  }


};

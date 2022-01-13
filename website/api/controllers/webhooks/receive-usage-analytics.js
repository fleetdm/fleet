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
  },


  exits: {
    success: { description: 'Analytics data was stored successfully.' },
  },


  fn: async function (inputs) {

    await HistoricalUsageSnapshot.create(inputs);

  }


};

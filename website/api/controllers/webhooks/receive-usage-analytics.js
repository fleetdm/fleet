module.exports = {


  friendlyName: 'Receive usage analytics',


  description: 'Receive anonymous usage analytics from deployments of Fleet running in production.  (Not fleetctl preview or dev-mode deployments.)',


  inputs: {
    anonymousIdentifier: { required: true, type: 'string', example: '9pnzNmrES3mQG66UQtd29cYTiX2+fZ4CYxDvh495720=', description: 'An anonymous identifier telling us which Fleet deployment this is.', },
    fleetVersion: { required: true, type: 'string', example: 'x.x.x' },
    licenseTier: { required: true, type: 'string', example: 'free' },
    numHostsEnrolled: { required: true, type: 'number', min: 0, custom: (num) => Math.floor(num) === num },
    numUsers: { required: true, type: 'number' },
    numTeams: { required: true, type: 'number' },
    numPolicies: { required: true, type: 'number' },
    numLabels: { required: true, type: 'number' },
    softwareInventoryEnabled: { required: true, type: 'boolean' },
    vulnDetectionEnabled: { required: true, type: 'boolean' },
    systemUsersEnabled: { required: true, type: 'boolean' },
    hostStatusWebhookEnabled: { required: true, type: 'boolean' },
  },


  exits: {
    success: { description: 'Analytics data was stored successfully.' },
  },


  fn: async function ({
    anonymousIdentifier,
    fleetVersion,
    licenseTier,
    numHostsEnrolled,
    numUsers,
    numTeams,
    numPolicies,
    softwareInventoryEnabled,
    vulnDetectionEnabled,
    systemUsersEnabled,
    hostStatusWebhookEnabled,
  }) {

    await HistoricalUsageSnapshot.create({
      anonymousIdentifier,
      fleetVersion,
      licenseTier,
      numHostsEnrolled,
      numUsers,
      numTeams,
      numPolicies,
      softwareInventoryEnabled,
      vulnDetectionEnabled,
      systemUsersEnabled,
      hostStatusWebhookEnabled,
    });

  }


};

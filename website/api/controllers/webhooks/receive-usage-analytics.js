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
    numWeeklyPolicyViolationDaysActual: { type: 'number', defaultsTo: 0 },
    numWeeklyPolicyViolationDaysPossible: { type: 'number', defaultsTo: 0 },
    hostsEnrolledByOperatingSystem: { type: {}, defaultsTo: {} },
    hostsEnrolledByOrbitVersion: { type: [{version: 'string', numEnrolled: 'number'}], defaultsTo: [] }, // TODO: The name of this parameter does not match naming conventions.
    hostsEnrolledByOsqueryVersion: { type: [{version: 'string', numEnrolled: 'number'}], defaultsTo: [] }, // TODO: The name of this parameter does not match naming conventions.
    storedErrors: { type: [{}], defaultsTo: [] }, // TODO migrate all rows that have "[]" to {}
    numHostsNotResponding: { type: 'number', defaultsTo: 0, description: 'The number of hosts per deployment that have not submitted results for distibuted queries. A host is counted as not responding if Fleet hasn\'t received a distributed write to requested distibuted queries for the host during the 2-hour interval since the host was last seen. Hosts that have not been seen for 7 days or more are not counted.', },
    organization: { type: 'string', defaultsTo: 'unknown', description: 'For Fleet Premium deployments, the organization registered with the license.', },
  },


  exits: {
    success: { description: 'Analytics data was stored successfully.' },
  },


  fn: async function (inputs) {

    let newUsageSnapshot = _.clone(inputs);

    // If hostsEnrolledByOperatingSystem has values
    if(inputs.hostsEnrolledByOperatingSystem !== {}) {
      // Create a new object that contains an empty totalHostsByOS object.
      hostsEnrolledByOperatingSystemWithTotals = _.extend({totalHostsByOS: {}}, inputs.hostsEnrolledByOperatingSystem);
      let totalHostsByOperatingSystem = {};
      // Iterate through the array of operating systems
      for(let operatingSystem in inputs.hostsEnrolledByOperatingSystem) {
        let totalNumberOfHostsUsingThisOperatingSystem = 0;
        // Iterate through array of operating system versions
        for(let osVersion of inputs.hostsEnrolledByOperatingSystem[operatingSystem]) {
          // Add the `numEnrolled` for this version to the totalNumberOfHostsUsingThisOperatingSystem
          totalNumberOfHostsUsingThisOperatingSystem += osVersion.numEnrolled;
        }
        // Add the totalNumberOfHostsUsingThisOperatingSystem value to the totalHostsByOperatingSystem object, using the name of the operating system as the key.
        totalHostsByOperatingSystem[operatingSystem] = totalNumberOfHostsUsingThisOperatingSystem;
      }
      hostsEnrolledByOperatingSystemWithTotals.totalHostsByOS = _.clone(totalHostsByOperatingSystem);

      // Add the hostsEnrolledByOperatingSystemWithTotals object to the newUsageSnapshot
      newUsageSnapshot.hostsEnrolledByOperatingSystem = _.clone(hostsEnrolledByOperatingSystemWithTotals);
    }

    await HistoricalUsageSnapshot.create(newUsageSnapshot);

  }


};

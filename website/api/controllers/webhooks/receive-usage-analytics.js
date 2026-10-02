// Fleet v4.93.0 and later report each Fleet-maintained app as an object instead of a slug. A missing
// flag means the reporting Fleet version didn't send it, which is not the same as `false`.
let isValidListOfFleetMaintainedApps = (apps)=>{
  if (!Array.isArray(apps)) {
    return false;
  }
  return apps.every((app)=>{
    if (typeof app === 'string') {
      return true;
    }
    if (!app || typeof app !== 'object' || Array.isArray(app) || typeof app.name !== 'string' || app.name === '') {
      return false;
    }
    return ['patchPolicy', 'softwareAutomation'].every((flag)=>{ return app[flag] === undefined || typeof app[flag] === 'boolean'; });
  });
};

// `null` is allowed so that a missing count can be stored as unknown instead of zero.
let isNullOrNonNegativeInteger = (num)=>{ return num === null || (Number.isInteger(num) && num >= 0); };

const OPTIONAL_COUNT_INPUT_NAMES = [
  'numPoliciesAutomationEnabledSoftware',
  'numMDMAppleProfiles',
  'numMDMWindowsProfiles',
  'numMDMAppleDeclarations',
  'numMDMAndroidProfiles',
];

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
    mdmAndroidEnabled: {type: 'boolean', defaultsTo: false},
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
    numHostsABMPending: {type: 'number', defaultsTo: 0 },
    fleetMaintainedAppsWindows: {type: 'json', defaultsTo: [], custom: isValidListOfFleetMaintainedApps, description: 'Fleet-maintained app slugs, or objects like {name, patchPolicy, softwareAutomation}.' },
    fleetMaintainedAppsMacOS: {type: 'json', defaultsTo: [], custom: isValidListOfFleetMaintainedApps, description: 'Fleet-maintained app slugs, or objects like {name, patchPolicy, softwareAutomation}.' },
    oktaConditionalAccessConfigured: {type: 'boolean', defaultsTo: false},
    entraConditionalAccessConfigured: {type: 'boolean', defaultsTo: false},
    conditionalAccessBypassDisabled: {type: 'boolean', defaultsTo: false},
    conditionalAccessEnabled: {type: 'boolean', defaultsTo: false},
    gitOpsModeEnabled: {type: 'boolean', defaultsTo: false},
    gitOpsModeExceptions: {type: ['string'], defaultsTo: [] },
    fleetDesktopSSOEnabled: {type: 'boolean', defaultsTo: false},
    numHostsFleetMDMEnrolledMacOS: {type: 'number', defaultsTo: 0 },
    numHostsFleetMDMEnrolledWindows: {type: 'number', defaultsTo: 0 },
    resultLogDestination: {type: 'string', defaultsTo: 'unknown'},
    statusLogDestination: {type: 'string', defaultsTo: 'unknown'},
    auditLogDestination: {type: 'string', defaultsTo: 'unknown'},
    anyVulnerabilitiesWebhookEnabled: {type: 'boolean', defaultsTo: false},
    anyFailingPoliciesWebhookEnabled: {type: 'boolean', defaultsTo: false},
    anyHostActivitiesWebhookEnabled: {type: 'boolean', defaultsTo: false},
    globalActivityWebhookEnabled: {type: 'boolean', defaultsTo: false},
    ticketDestinationConfigured: {type: 'boolean', defaultsTo: false},
    ssoConfiguredFleetUsers: {type: 'boolean', defaultsTo: false},
    ssoConfiguredEndUsers: {type: 'boolean', defaultsTo: false},
    accountProvisioningConfigured: {type: 'boolean', defaultsTo: false},
    idpSCIMConfigured: {type: 'boolean', defaultsTo: false},
    idpGoogleWorkspaceConfigured: {type: 'boolean', defaultsTo: false},
    certificateAuthorityConfigured: {type: 'boolean', defaultsTo: false},
    // No defaults: older Fleet versions don't report these, and that must stay distinguishable from zero.
    numPoliciesAutomationEnabledSoftware: {type: 'number', allowNull: true, custom: isNullOrNonNegativeInteger},
    numMDMAppleProfiles: {type: 'number', allowNull: true, custom: isNullOrNonNegativeInteger},
    numMDMWindowsProfiles: {type: 'number', allowNull: true, custom: isNullOrNonNegativeInteger},
    numMDMAppleDeclarations: {type: 'number', allowNull: true, custom: isNullOrNonNegativeInteger},
    numMDMAndroidProfiles: {type: 'number', allowNull: true, custom: isNullOrNonNegativeInteger},
  },


  exits: {
    success: { description: 'Analytics data was stored successfully.' },
  },


  fn: async function (inputs) {
    // Empty strings would fail the model's `required` validation, so fall back to the default value.
    for(let stringInput of ['organization', 'resultLogDestination', 'statusLogDestination', 'auditLogDestination']) {
      if(inputs[stringInput] === '') {
        inputs[stringInput] = 'unknown';
      }
    }
    // Set omitted counts to null explicitly rather than relying on Waterline, which can fill in a type's base value (0).
    for(let countInput of OPTIONAL_COUNT_INPUT_NAMES) {
      if(inputs[countInput] === undefined) {
        inputs[countInput] = null;
      }
    }
    // Create a database record for these usage statistics.
    await HistoricalUsageSnapshot.create(Object.assign({}, inputs));

  }


};

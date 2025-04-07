/**
 * HistoricalUsageSnapshot.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
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
    hostsStatusWebHookEnabled: { required: true, type: 'boolean'},
    numWeeklyActiveUsers: { required: true, type: 'number' },
    numWeeklyPolicyViolationDaysActual: { required: true, type: 'number' },
    numWeeklyPolicyViolationDaysPossible: { required: true, type: 'number'},
    hostsEnrolledByOperatingSystem: { required: true, type: 'json' },
    hostsEnrolledByOrbitVersion: { required: true, type: 'json' },
    hostsEnrolledByOsqueryVersion: { required: true, type: 'json' },
    storedErrors: { required: true, type: 'json' },
    numHostsNotResponding: { required: true, type: 'number', description: 'The number of hosts per deployment that have not submitted results for distibuted queries. A host is counted as not responding if Fleet hasn\'t received a distributed write to requested distibuted queries for the host during the 2-hour interval since the host was last seen. Hosts that have not been seen for 7 days or more are not counted.', },
    organization: { required: true, type: 'string' },
    mdmMacOsEnabled: {required: true, type: 'boolean'},
    mdmWindowsEnabled: {required: true, type: 'boolean'},
    liveQueryDisabled: {required: true, type: 'boolean'},
    hostExpiryEnabled: {required: true, type: 'boolean'},
    numSoftwareVersions: {required: true, type: 'number'},
    numHostSoftwares: {required: true, type: 'number'},
    numSoftwareTitles: {required: true, type: 'number'},
    numHostSoftwareInstalledPaths: {required: true, type: 'number'},
    numSoftwareCPEs: {required: true, type: 'number'},
    numSoftwareCVEs: {required: true, type: 'number'},
    aiFeaturesDisabled: {required: true, type: 'boolean'},
    maintenanceWindowsEnabled: {required: true, type: 'boolean'},
    maintenanceWindowsConfigured: {required: true, type: 'boolean'},
    numHostsFleetDesktopEnabled: {required: true, type: 'number'},
    numQueries: {required: true, type: 'number' },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

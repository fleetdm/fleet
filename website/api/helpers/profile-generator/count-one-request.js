module.exports = {


  friendlyName: 'Count one request',


  description: 'Count one configuration profile generator request against the monthly limit of a Fleet Free registration or a Fleet Premium license.',


  extendedDescription: 'The counters are changed with a compare-and-set: an update that only applies if they still hold the values just read, retried if another request changed them first.  This keeps concurrent requests from going past the limit without a database-specific increment.',


  inputs: {

    registrationId: {
      type: 'number',
      description: 'The Fleet Free registration to count against.  Exactly one of this and `licenseUsageId` is required.',
    },

    licenseUsageId: {
      type: 'number',
      description: 'The usage row of the Fleet Premium license to count against.',
    },

    requestType: {
      type: 'string',
      required: true,
      isIn: ['profile', 'nameAndDescription'],
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Usage',
      outputDescription: 'The UTC month the request was counted in, how many of each request type the registration or license has now made in it, and its monthly profile limit.',
      outputType: {
        usageMonth: 'string',
        profileGenerationsThisMonth: 'number',
        nameAndDescriptionRequestsThisMonth: 'number',
        profileLimit: 'number',
      },
    },

    limitReached: {
      description: 'This registration or license has reached its limit for the month.',
      outputFriendlyName: 'Limit',
      outputDescription: 'When the limit resets, and the monthly profile limit (even when it was the name and description limit that was reached).',
      outputType: {
        resetsAt: 'string',
        profileLimit: 'number',
      },
    },

  },


  fn: async function ({registrationId, licenseUsageId, requestType}) {

    if(!registrationId === !licenseUsageId) {
      throw new Error('Could not count a configuration profile generator request: exactly one of registrationId and licenseUsageId is required.');
    }
    let model = registrationId ? ProfileGeneratorRegistration : ProfileGeneratorLicenseUsage;
    let id = registrationId || licenseUsageId;
    let limits = registrationId ? sails.config.profileGeneratorLimits.fleetFree : sails.config.profileGeneratorLimits.fleetPremium;
    let profileLimit = limits.profiles;
    let nameAndDescriptionLimit = limits.namesAndDescriptions;

    // Each retry means another request changed the counters in between, so this only runs out under a flood
    // of concurrent requests with one credential.
    const MAX_ATTEMPTS = 5;
    for(let attempt = 0; attempt < MAX_ATTEMPTS; attempt++) {
      let record = await model.findOne({ id });
      if(!record) {
        throw new Error(`Could not count a configuration profile generator request: no ${model.identity} has the id ${id}.`);
      }

      // Read after the row, so a request that started before midnight UTC and reads a row a newer request
      // already moved into the new month counts in that month rather than moving it back.  YYYY-MM sorts by date.
      let currentMonth = new Date().toISOString().slice(0, 7);
      let thisMonth = record.usageMonth > currentMonth ? record.usageMonth : currentMonth;
      let [year, month] = thisMonth.split('-').map(Number);
      let resetsAt = new Date(Date.UTC(year, month, 1)).toISOString();

      // The first request in a new month starts the counters from zero.
      let isSameMonth = record.usageMonth === thisMonth;
      let profileGenerations = isSameMonth ? record.profileGenerationsThisMonth : 0;
      let nameAndDescriptionRequests = isSameMonth ? record.nameAndDescriptionRequestsThisMonth : 0;

      // A name and description is only useful next to a profile, so it stops when profiles do.
      if(profileGenerations >= profileLimit || (requestType === 'nameAndDescription' && nameAndDescriptionRequests >= nameAndDescriptionLimit)) {
        throw {limitReached: {resetsAt, profileLimit}};
      }

      let usage = {
        usageMonth: thisMonth,
        profileGenerationsThisMonth: profileGenerations + (requestType === 'profile' ? 1 : 0),
        nameAndDescriptionRequestsThisMonth: nameAndDescriptionRequests + (requestType === 'nameAndDescription' ? 1 : 0),
      };
      let updatedRecord = await model.updateOne({
        id,
        usageMonth: record.usageMonth,
        profileGenerationsThisMonth: record.profileGenerationsThisMonth,
        nameAndDescriptionRequestsThisMonth: record.nameAndDescriptionRequestsThisMonth,
      }).set(usage);
      if(updatedRecord) {
        return _.extend({ profileLimit }, usage);
      }
    }
    throw new Error(`Could not count a configuration profile generator request for ${model.identity} ${id} after ${MAX_ATTEMPTS} attempts, because other requests kept changing its counters.`);

  }


};

module.exports = {


  friendlyName: 'Count one request',


  description: 'Count one configuration profile generator request against a Fleet Free registration\'s monthly limit.',


  extendedDescription: 'The counters are changed with a compare-and-set: an update that only applies if they still hold the values just read, retried if another request changed them first.  This keeps concurrent requests from going past the limit without a database-specific increment.',


  inputs: {

    registrationId: {
      type: 'number',
      required: true,
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
      outputDescription: 'The UTC month the request was counted in, and how many of each request type the registration has now made in it.',
      outputType: {
        usageMonth: 'string',
        profileGenerationsThisMonth: 'number',
        nameAndDescriptionRequestsThisMonth: 'number',
      },
    },

    limitReached: {
      description: 'This registration has reached its limit for the month.',
      outputFriendlyName: 'When the limit resets',
      outputType: 'string',
    },

  },


  fn: async function ({registrationId, requestType}) {

    let profileLimit = sails.config.custom.profileGeneratorMonthlyProfileLimit;
    let nameAndDescriptionLimit = sails.config.custom.profileGeneratorMonthlyNameAndDescriptionLimit;

    // Each retry means another request changed the counters in between, so this only runs out under a flood
    // of concurrent requests from one registration.
    const MAX_ATTEMPTS = 20;
    for(let attempt = 0; attempt < MAX_ATTEMPTS; attempt++) {
      let registration = await ProfileGeneratorRegistration.findOne({ id: registrationId });
      if(!registration) {
        throw new Error(`Could not count a configuration profile generator request: no registration has the id ${registrationId}.`);
      }

      // Read after the row, so a request that started before midnight UTC and reads a row a newer request
      // already moved into the new month counts in that month rather than moving it back.  YYYY-MM sorts by date.
      let currentMonth = new Date().toISOString().slice(0, 7);
      let thisMonth = registration.usageMonth > currentMonth ? registration.usageMonth : currentMonth;
      let [year, month] = thisMonth.split('-').map(Number);
      let resetsAt = new Date(Date.UTC(year, month, 1)).toISOString();

      // The first request in a new month starts the counters from zero.
      let isSameMonth = registration.usageMonth === thisMonth;
      let profileGenerations = isSameMonth ? registration.profileGenerationsThisMonth : 0;
      let nameAndDescriptionRequests = isSameMonth ? registration.nameAndDescriptionRequestsThisMonth : 0;

      // A name and description is only useful next to a profile, so it stops when profiles do.
      if(profileGenerations >= profileLimit || (requestType === 'nameAndDescription' && nameAndDescriptionRequests >= nameAndDescriptionLimit)) {
        throw {limitReached: resetsAt};
      }

      let usage = {
        usageMonth: thisMonth,
        profileGenerationsThisMonth: profileGenerations + (requestType === 'profile' ? 1 : 0),
        nameAndDescriptionRequestsThisMonth: nameAndDescriptionRequests + (requestType === 'nameAndDescription' ? 1 : 0),
      };
      let updatedRegistration = await ProfileGeneratorRegistration.updateOne({
        id: registrationId,
        usageMonth: registration.usageMonth,
        profileGenerationsThisMonth: registration.profileGenerationsThisMonth,
        nameAndDescriptionRequestsThisMonth: registration.nameAndDescriptionRequestsThisMonth,
      }).set(usage);
      if(updatedRegistration) {
        return usage;
      }
    }
    throw new Error(`Could not count a configuration profile generator request for registration ${registrationId} after ${MAX_ATTEMPTS} attempts, because other requests kept changing its counters.`);

  }


};

module.exports = {


  friendlyName: 'Give back one request',


  description: 'Undo counting one configuration profile generator request, for a request the AI service never answered.',


  extendedDescription: 'Uses the same compare-and-set as counting.  Nothing is given back once the month the request was counted in has ended, since the counters then belong to a new month.',


  inputs: {

    registrationId: {
      type: 'number',
      description: 'The Fleet Free registration the request was counted against.  Exactly one of this and `licenseUsageId` is required.',
    },

    licenseUsageId: {
      type: 'number',
      description: 'The usage row of the Fleet Premium license the request was counted against.',
    },

    requestType: {
      type: 'string',
      required: true,
      isIn: ['profile', 'nameAndDescription'],
    },

    usageMonth: {
      type: 'string',
      required: true,
      description: 'The UTC month the request was counted in, as YYYY-MM.',
    },

  },


  fn: async function ({registrationId, licenseUsageId, requestType, usageMonth}) {

    if(!registrationId === !licenseUsageId) {
      throw new Error('Could not give back a configuration profile generator request: exactly one of registrationId and licenseUsageId is required.');
    }
    let model = registrationId ? ProfileGeneratorRegistration : ProfileGeneratorLicenseUsage;
    let id = registrationId || licenseUsageId;

    let counterName = requestType === 'profile' ? 'profileGenerationsThisMonth' : 'nameAndDescriptionRequestsThisMonth';

    const MAX_ATTEMPTS = 5;
    for(let attempt = 0; attempt < MAX_ATTEMPTS; attempt++) {
      let record = await model.findOne({ id });
      if(!record || record.usageMonth !== usageMonth || record[counterName] <= 0) {
        return;
      }
      let updatedRecord = await model.updateOne({
        id,
        usageMonth,
        [counterName]: record[counterName],
      }).set({ [counterName]: record[counterName] - 1 });
      if(updatedRecord) {
        return;
      }
    }
    sails.log.warn(`Could not give back a ${requestType} request to configuration profile generator ${model.identity} ${id} after ${MAX_ATTEMPTS} attempts, because other requests kept changing its counters.`);

  }


};

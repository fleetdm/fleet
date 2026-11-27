module.exports = {


  friendlyName: 'Give back one request',


  description: 'Undo counting one configuration profile generator request, for a request the AI service never answered.',


  extendedDescription: 'Uses the same compare-and-set as counting.  Nothing is given back once the month the request was counted in has ended, since the counters then belong to a new month.',


  inputs: {

    usageId: {
      type: 'number',
      required: true,
      description: 'The ProfileGeneratorUsage record the request was counted against.',
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


  fn: async function ({usageId, requestType, usageMonth}) {

    let counterName = requestType === 'profile' ? 'profileGenerationsThisMonth' : 'nameAndDescriptionRequestsThisMonth';

    const MAX_ATTEMPTS = 5;
    for(let attempt = 0; attempt < MAX_ATTEMPTS; attempt++) {
      let record = await ProfileGeneratorUsage.findOne({ id: usageId });
      if(!record || record.usageMonth !== usageMonth || record[counterName] <= 0) {
        return;
      }
      let updatedRecord = await ProfileGeneratorUsage.updateOne({
        id: usageId,
        usageMonth,
        [counterName]: record[counterName],
      }).set({ [counterName]: record[counterName] - 1 });
      if(updatedRecord) {
        return;
      }
    }
    sails.log.warn(`Could not give back a ${requestType} request to configuration profile generator usage record ${usageId} after ${MAX_ATTEMPTS} attempts, because other requests kept changing its counters.`);

  }


};

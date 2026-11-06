module.exports = {


  friendlyName: 'Give back one request',


  description: 'Undo counting one configuration profile generator request, for a request the AI service never answered.',


  extendedDescription: 'Uses the same compare-and-set as counting.  Nothing is given back once the month the request was counted in has ended, since the counters then belong to a new month.',


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

    usageMonth: {
      type: 'string',
      required: true,
      description: 'The UTC month the request was counted in, as YYYY-MM.',
    },

  },


  fn: async function ({registrationId, requestType, usageMonth}) {

    let counterName = requestType === 'profile' ? 'profileGenerationsThisMonth' : 'nameAndDescriptionRequestsThisMonth';

    const MAX_ATTEMPTS = 5;
    for(let attempt = 0; attempt < MAX_ATTEMPTS; attempt++) {
      let registration = await ProfileGeneratorRegistration.findOne({ id: registrationId });
      if(!registration || registration.usageMonth !== usageMonth || registration[counterName] <= 0) {
        return;
      }
      let updatedRegistration = await ProfileGeneratorRegistration.updateOne({
        id: registrationId,
        usageMonth,
        [counterName]: registration[counterName],
      }).set({ [counterName]: registration[counterName] - 1 });
      if(updatedRegistration) {
        return;
      }
    }
    sails.log.warn(`Could not give back a ${requestType} request to configuration profile generator registration ${registrationId} after ${MAX_ATTEMPTS} attempts, because other requests kept changing its counters.`);

  }


};

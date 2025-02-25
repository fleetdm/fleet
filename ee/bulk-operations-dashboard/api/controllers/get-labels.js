module.exports = {


  friendlyName: 'Get labels',


  description: 'Builds and returns an array of labels on the Fleet instance.',


  exits: {
    success: {
      outputType: [{}],
    }
  },


  fn: async function () {


    let labelsOnThisInstance = [];

    let labelsResponseData = await sails.helpers.http.get.with({
      url: '/api/v1/fleet/labels',
      baseUrl: sails.config.custom.fleetBaseUrl,
      headers: {
        Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
      }
    })
    .timeout(120000)
    .retry(['requestFailed', {name: 'TimeoutError'}]);

    for(let label of labelsResponseData.labels) {
      labelsOnThisInstance.push({
        name: label.name,
        value: label.id
      });
    }
    labelsOnThisInstance = _.sortBy(labelsOnThisInstance, 'name');
    // All done.
    return labelsOnThisInstance;

  }


};

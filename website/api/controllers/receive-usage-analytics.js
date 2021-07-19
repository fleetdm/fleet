module.exports = {


  friendlyName: 'Receive usage analytics',


  description: '',


  inputs: {
    anonymousIdentifier: { required: true, type: 'string', example: '1', description: 'An anonymous identifier telling us which Fleet deployment this is.', },
    fleetVersion: { required: true, type: 'string', example: 'x.x.x' },
    numHostsEnrolled: { required: true, type: 'number', min: 0, custom: (num) => Math.floor(num) === num },
  },


  exits: {
    success: { description: 'Analytics data was stored successfully.' },
  },


  fn: async function ({anonymousIdentifier, fleetVersion, numHostsEnrolled}) {

    await HistoricalUsageSnapshot.create({
      anonymousIdentifier,
      fleetVersion,
      numHostsEnrolled
    });

  }


};

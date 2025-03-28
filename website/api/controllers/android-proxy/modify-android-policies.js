module.exports = {


  friendlyName: 'Modify android policies',


  description: '',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    profileId: {
      type: 'string',
      required: true,
    },
    fleetServerSecret: {
      type: 'string',
    },
    policy: {
      type: {},
      moreInfoUrl: 'https://developers.google.com/android/management/reference/rest/v1/enterprises.policies#Policy'
    }
  },


  exits: {

  },


  fn: async function ({androidEnterpriseId, profileId, fleetServerSecret}) {


    //fleetdm.com/api/v1/android/:androidEnterpriseId/configuration-profiles/:profileId`
    // All done.
    return;

  }


};

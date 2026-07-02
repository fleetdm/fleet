module.exports = {


  friendlyName: 'Get is enterprise managed by fleet',


  description: 'Checks the list of Android Enterprises managed by Fleet\'s Enterprise Management Google project and returns true if the provided enterprise ID is present.',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
      description: 'The enterprise ID of the Android Enterprise '
    }
  },


  exits: {

    success: {
      outputFriendlyName: 'Is enterprise managed by fleet',
      outputType: 'boolean',
    },
  },


  fn: async function ({androidEnterpriseId}) {

    require('assert')(sails.config.custom.androidEnterpriseServiceAccountEmailAddress);
    require('assert')(sails.config.custom.androidEnterpriseServiceAccountPrivateKey);
    require('assert')(sails.config.custom.androidEnterpriseProjectId);

    let isEnterpriseManagedByFleet = false;

    // Log into google.
    // Reuse the shared Google API auth client created at server startup (see api/hooks/custom/).
    let { google } = require('googleapis');
    let androidmanagement = google.androidmanagement({version: 'v1', auth: sails.googleAuthClient});

    // Use Google's LIST call to check if enterprise exists.
    let enterprises = [];
    let tokenForNextPageOfEnterprises;
    await sails.helpers.flow.until(async ()=>{
      let listEnterprisesResponse = await androidmanagement.enterprises.list({
        projectId: sails.config.custom.androidEnterpriseProjectId,
        pageSize: 100,
        pageToken: tokenForNextPageOfEnterprises,
      });
      tokenForNextPageOfEnterprises = listEnterprisesResponse.data.nextPageToken;
      enterprises = enterprises.concat(listEnterprisesResponse.data.enterprises);

      if(!listEnterprisesResponse.data.nextPageToken){
        return true;
      }
    });

    // Check the list of enterprises
    let enterpriseExistsInTheListOfEnterprises = _.find(enterprises, (enterprise)=>{
      return enterprise.name === `enterprises/${androidEnterpriseId}` || enterprise.name === androidEnterpriseId;
    });

    if(enterpriseExistsInTheListOfEnterprises){
      isEnterpriseManagedByFleet = true;
    }

    // Send back the result through the success exit.
    return isEnterpriseManagedByFleet;
  }


};


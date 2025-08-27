module.exports = {


  friendlyName: 'Get android enterprises',


  description: 'List all android enterprises accessible to the calling user.',


  inputs: {
    // No inputs needed for list endpoint
  },


  exits: {
    success: { description: 'Android enterprises list was successfully retrieved.' }
  },


  fn: async function () {

    // Extract fleetServerSecret from the Authorization header
    let authHeader = this.req.get('authorization');

    if (authHeader && authHeader.startsWith('Bearer')) {
      // We extract the token for validation but don't need to use it for LIST endpoint
    } else {
      return this.res.unauthorized('Authorization header with Bearer token is required');
    }

    // Get the Android enterprises list from Google
    try {
      let enterprisesList = await sails.helpers.flow.build(async ()=>{
        let { google } = require('googleapis');
        let androidmanagement = google.androidmanagement('v1');

        let googleAuth = new google.auth.GoogleAuth({
          scopes: [
            'https://www.googleapis.com/auth/androidmanagement'
          ],
          credentials: {
            client_email: sails.config.custom.androidEnterpriseServiceAccountEmailAddress,// eslint-disable-line camelcase
            private_key: sails.config.custom.androidEnterpriseServiceAccountPrivateKey,// eslint-disable-line camelcase
          },
        });
        
        // Acquire the google auth client, and bind it to all future calls
        let authClient = await googleAuth.getClient();
        google.options({auth: authClient});

        // List all enterprises accessible to this service account
        let listEnterprisesResponse = await androidmanagement.enterprises.list({
          projectId: sails.config.custom.androidEnterpriseProjectId,
        });
        
        return listEnterprisesResponse.data;
      }).intercept((err)=>{
        // Re-throw the error for handling outside the intercept
        throw err;
      });

      // Return the enterprises list (or empty list if no enterprises)
      return enterprisesList || { enterprises: [] };

    } catch (err) {
      throw new Error(`When attempting to list android enterprises, an error occurred. Error: ${err}`);
    }

  }


};

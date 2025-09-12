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
    let fleetServerSecret;

    if (authHeader && authHeader.startsWith('Bearer')) {
      fleetServerSecret = authHeader.replace('Bearer', '').trim();
    } else {
      return this.res.unauthorized('Authorization header with Bearer token is required');
    }

    let fleetServerUrl = this.req.get('Origin');
    if (!fleetServerUrl) {
      return this.res.badRequest('Origin header is required');
    }

    let thisAndroidEnterprise = await AndroidEnterprise.findOne({
      fleetServerUrl: fleetServerUrl
    });

    if (!thisAndroidEnterprise) {
      return this.res.notFound('No Android enterprise found for this Fleet server');
    }

    if (thisAndroidEnterprise.fleetServerSecret !== fleetServerSecret) {
      return this.res.unauthorized('Invalid authentication token');
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
        let allEnterprises = [];
        let tokenForNextPageOfEnterprises;
        await sails.helpers.flow.until(async ()=>{
          let listEnterprisesResponse = await androidmanagement.enterprises.list({
            projectId: sails.config.custom.androidEnterpriseProjectId,
            pageSize: 100,
            pageToken: tokenForNextPageOfEnterprises,
          });
          tokenForNextPageOfEnterprises = listEnterprisesResponse.data.nextPageToken;
          if (listEnterprisesResponse.data.enterprises) {
            allEnterprises = allEnterprises.concat(listEnterprisesResponse.data.enterprises);
          }

          if(!listEnterprisesResponse.data.nextPageToken){
            return true;
          }
        });

        return allEnterprises;
      }).intercept((err)=>{
        // Re-throw the error for handling outside the intercept
        return err;
      });

      // Filter the results to only include enterprises belonging to this Fleet instance
      let allEnterprises = enterprisesList || [];
      let filteredEnterprises = allEnterprises.filter(enterprise => {
        if (!enterprise) {
          return false;
        }
        // Extract enterprise ID from the Google enterprise name (format: "enterprises/ENTERPRISE_ID")
        let enterpriseId = enterprise.name ? enterprise.name.split('/')[1] : null;
        return enterpriseId === thisAndroidEnterprise.androidEnterpriseId;
      });

      // Return only the enterprises belonging to this Fleet instance
      return { enterprises: filteredEnterprises };

    } catch (err) {
      throw new Error(`When attempting to list android enterprises, an error occurred. Error: ${err}`);
    }

  }


};

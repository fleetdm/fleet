module.exports = {


  friendlyName: 'Get one android enterprise',


  description: 'Get details about an android enterprise.',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: 'Android enterprise details were successfully retrieved.' }
  },


  fn: async function ({androidEnterpriseId}) {

    // Extract fleetServerSecret from the Authorization header
    let authHeader = this.req.get('authorization');
    let fleetServerSecret;

    if (authHeader && authHeader.startsWith('Bearer')) {
      fleetServerSecret = authHeader.replace('Bearer', '').trim();
    } else {
      return this.res.unauthorized('Authorization header with Bearer token is required');
    }

    let thisAndroidEnterprise = await AndroidEnterprise.findOne({
      androidEnterpriseId: androidEnterpriseId,
    });

    // If not found by androidEnterpriseId, try to find by fleetServerSecret
    if (!thisAndroidEnterprise) {
      thisAndroidEnterprise = await AndroidEnterprise.findOne({
        fleetServerSecret: fleetServerSecret
      });
      
      // If found by fleetServerSecret, verify the androidEnterpriseId matches
      if (thisAndroidEnterprise && thisAndroidEnterprise.androidEnterpriseId !== androidEnterpriseId) {
        // This is likely a 403 case - enterprise exists but with different ID
        return this.res.forbidden('Enterprise access forbidden');
      }
    }

    // If still no enterprise found, return 403 instead of trying Fleet backend fallback
    if(!thisAndroidEnterprise) {
      return this.res.forbidden('Enterprise not found or access denied');
    }


    // Return an unauthorized response if the provided secret does not match.
    if(thisAndroidEnterprise.fleetServerSecret !== fleetServerSecret) {
      return this.res.unauthorized();
    }

    // Get the Android enterprise details from Google (normal flow with proxy DB record)
    // Note: We're using sails.helpers.flow.build here to handle any errors that occur using google's node library.
    try {
      let enterpriseDetails = await sails.helpers.flow.build(async ()=>{
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

        // Get the android enterprise details.
        let getEnterpriseResponse = await androidmanagement.enterprises.get({
          name: `enterprises/${androidEnterpriseId}`,
        });
        
        return getEnterpriseResponse.data;
      }).intercept((err)=>{
        // Return the error for handling outside the intercept
        // Note: In Sails.js, .intercept() handlers should return errors, not throw them
        return err;
      });

      // Return the enterprise details
      return enterpriseDetails;
      
    } catch (err) {
      // Handle Google API errors properly here
      if (err.status === 403 || (err.errors && err.errors.some(e => e.reason === 'forbidden'))) {
        return this.res.notFound('Android Enterprise has been deleted or is no longer accessible');
      }
      
      throw new Error(`When attempting to get android enterprise details (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    }

  }


};

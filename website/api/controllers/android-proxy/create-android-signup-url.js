module.exports = {


  friendlyName: 'Create android signup url',


  description: 'Creates and returns a signup URL for an android enterprise.',


  inputs: {
    callbackUrl: {
      type: 'string',
      required: true,
    }
  },


  exits: {
    success: { description: 'A signup URL has been sent to the requesting Fleet server.'},
    enterpriseAlreadyExists: { description: 'An Android enterprise already exists for this Fleet instance.', statusCode: 409 },
    invalidCallbackUrl: { description: 'The provided callbackUrl could not be used to create an Android enterprise signup URL.', responseType: 'badRequest'}
  },


  fn: async function ({ callbackUrl }) {


    // Parse the Fleet server url from the origin header.
    let fleetServerUrl = this.req.get('Origin');
    if(!fleetServerUrl){
      return this.res.badRequest();
    }

    // Check the database for an existing record for this Fleet server.
    let connectionforThisInstanceExists = await AndroidEnterprise.findOne({fleetServerUrl: fleetServerUrl});
    if(connectionforThisInstanceExists){
      // Before throwing conflict, verify the enterprise still exists in Google
      // If it doesn't exist, clean up the stale proxy record and continue with signup
      try {
        let { google } = require('googleapis');
        let androidmanagement = google.androidmanagement('v1');
        let googleAuth = new google.auth.GoogleAuth({
          scopes: ['https://www.googleapis.com/auth/androidmanagement'],
          credentials: {
            client_email: sails.config.custom.androidEnterpriseServiceAccountEmailAddress,// eslint-disable-line camelcase
            private_key: sails.config.custom.androidEnterpriseServiceAccountPrivateKey,// eslint-disable-line camelcase
          },
        });
        let authClient = await googleAuth.getClient();
        google.options({auth: authClient});

        // Try to get the enterprise from Google
        await androidmanagement.enterprises.get({
          name: `enterprises/${connectionforThisInstanceExists.androidEnterpriseId}`,
        });

        // If we got here, enterprise still exists in Google - throw conflict
        throw 'enterpriseAlreadyExists';

      } catch (err) {
        // Only clean up proxy record if this is a Google API error indicating enterprise deletion
        if (err && err.code && (err.code === 403 || err.code === 404)) {
          // 403 or 404 from Google API indicates enterprise doesn't exist - clean up stale proxy record
          await AndroidEnterprise.destroyOne({ id: connectionforThisInstanceExists.id });
          // Continue with signup process (don't throw conflict)
        } else {
          // For other errors (network, auth, etc), re-throw to let them bubble up
          throw err;
        }
      }
    }


    // Get a signup url for this Android enterprise.
    // Note: We're using sails.helpers.flow.build here to handle any errors that occurr using google's node library.
    let signupUrl = await sails.helpers.flow.build(async ()=>{
      let { google } = require('googleapis');
      let androidmanagement = google.androidmanagement('v1');
      let googleAuth = new google.auth.GoogleAuth({
        scopes: ['https://www.googleapis.com/auth/androidmanagement'],
        credentials: {
          client_email: sails.config.custom.androidEnterpriseServiceAccountEmailAddress,// eslint-disable-line camelcase
          private_key: sails.config.custom.androidEnterpriseServiceAccountPrivateKey,// eslint-disable-line camelcase
        },
      });
      // Acquire the google auth client, and bind it to all future calls
      let authClient = await googleAuth.getClient();
      google.options({auth: authClient});
      // [?] https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Signupurls.html#create
      let createSignupUrlResponse = await androidmanagement.signupUrls.create({
        // The callback URL that the admin will be redirected to after successfully creating an enterprise. Before redirecting there the system will add a query parameter to this URL named enterpriseToken which will contain an opaque token to be used for the create enterprise request. The URL will be parsed then reformatted in order to add the enterpriseToken parameter, so there may be some minor formatting changes.
        callbackUrl: callbackUrl,
        // The ID of the Google Cloud Platform project which will own the enterprise.
        projectId: sails.config.custom.androidEnterpriseProjectId,
      });
      return createSignupUrlResponse.data;
    }).intercept({status: 400}, (unusedErr)=>{
      return {'invalidCallbackUrl': 'The provided Callback Url could not be used to create an Android enterprise signup URL.'};
    }).intercept((err)=>{
      return new Error(`When attempting to create a singup url for a new Android enterprise, an error occurred. Error: ${err}`);
    });


    return {
      url: signupUrl.url,
      name: signupUrl.name,
    };



  }


};

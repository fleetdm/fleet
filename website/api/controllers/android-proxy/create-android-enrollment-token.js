module.exports = {


  friendlyName: 'Create android enrollment token',


  description: 'Creates and returns an enrollment token for an Android enterprise',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
  },


  exits: {

  },


  fn: async function ({androidEnterpriseId}) {
    // Extract fleetServerSecret from the Authorization header
    let authHeader = this.req.headers.authorization;
    let fleetServerSecret;

    if (authHeader && authHeader.startsWith('Bearer')) {
      fleetServerSecret = authHeader.replace('Bearer', '').trim();
    } else {
      return this.res.unauthorized('Authorization header with Bearer token is required');
    }

    // Authenticate this request
    let thisAndroidEnterprise = await AndroidEnterprise.findOne({
      androidEnterpriseId: androidEnterpriseId,
    });

    // Return a 404 response if no records are found.
    if(!thisAndroidEnterprise) {
      return this.res.notFound();
    }

    // Return an unauthorized response if the provided secret does not match.
    if(thisAndroidEnterprise.fleetServerSecret !== fleetServerSecret) {
      return this.res.unauthorized();
    }

    let newEnrollmentToken = await sails.helpers.flow.build(async ()=>{
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
      // [?]: https://googleapis.dev/nodejs/googleapis/latest/androidmanagement/classes/Resource$Enterprises$Enrollmenttokens.html#create
      let enrollmentTokenCreateResponse = await androidmanagement.enterprises.enrollmentTokens.create({
        parent: `enterprises/${androidEnterpriseId}`,
        requestBody: this.req.body,
      });
      return enrollmentTokenCreateResponse.data;
    }).intercept((err)=>{
      return new Error(`When attempting to create an enrollment token for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    });


    return newEnrollmentToken;

  }


};

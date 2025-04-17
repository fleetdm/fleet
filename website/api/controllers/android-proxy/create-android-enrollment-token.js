module.exports = {


  friendlyName: 'Create android enrollment token',


  description: 'Creates and returns an enrollment token for an Android enterprise',


  inputs: {
    fleetServerSecret: {
      type: 'string',
      required: true,
    },
    androidEnterpriseId: {
      type: 'string',
      required: true,
    },
    enrollmentToken: {
      type: {},
      required: true,
      moreInfoUrl: 'https://developers.google.com/android/management/reference/rest/v1/enterprises.enrollmentTokens#EnrollmentToken',
    }
  },


  exits: {

  },


  fn: async function ({fleetServerSecret, androidEnterpriseId, enrollmentToken}) {
    // Authenticate this request
    let thisAndroidEnterprise = await AndroidEnterprise.findOne({
      fleetServerSecret: fleetServerSecret,
      androidEnterpriseId: androidEnterpriseId,
    });

    // Return a 404 response if no records are found.
    if(!thisAndroidEnterprise) {
      return this.res.notFound();
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
        requestBody: enrollmentToken,
      });
      return enrollmentTokenCreateResponse.data;
    }).intercept((err)=>{
      return new Error(`When attempting to create an enrollment token for an Android enterprise (${androidEnterpriseId}), an error occurred. Error: ${err}`);
    });


    return newEnrollmentToken;

  }


};

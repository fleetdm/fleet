module.exports = {


  friendlyName: 'Get connection',


  description: 'Return an authenticated jsforce connection to Fleet\'s Salesforce instance.',


  extendedDescription: 'This uses the OAuth 2.0 client credentials flow, because Salesforce is retiring SOAP API login() (the default jsforce login method) in the Summer \'27 release.',


  exits: {

    success: {
      outputFriendlyName: 'Salesforce connection',
      outputType: 'ref',
    },

  },


  fn: async function () {

    require('assert')(sails.config.custom.salesforceIntegrationClientId);
    require('assert')(sails.config.custom.salesforceIntegrationClientSecret);

    let jsforce = require('jsforce');
    let salesforceConnection = new jsforce.Connection({
      // The client credentials flow requires the My Domain URL; login.salesforce.com is not supported.
      loginUrl: 'https://fleetdm.my.salesforce.com',
      oauth2: {
        clientId: sails.config.custom.salesforceIntegrationClientId,
        clientSecret: sails.config.custom.salesforceIntegrationClientSecret,
      },
      // The client credentials flow does not return a refresh token, so request a new access token if the session expires.
      refreshFn: async (connection, callback)=>{
        try {
          await connection.authorize({ grant_type: 'client_credentials' });
          callback(undefined, connection.accessToken);
        } catch (err) {
          callback(err);
        }
      },
    });

    await sails.helpers.flow.build(async ()=>{
      await salesforceConnection.authorize({ grant_type: 'client_credentials' });
    }).intercept((err)=>{
      return new Error(`When attempting to authenticate with Salesforce using the OAuth 2.0 client credentials flow, an error occurred. Full error: ${require('util').inspect(err)}`);
    });

    return salesforceConnection;

  }


};

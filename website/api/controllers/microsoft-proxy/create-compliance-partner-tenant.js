module.exports = {


  friendlyName: 'Create compliance partner tenant',


  description: 'Creates a new Microsoft compliance partner tenant record for a provided tenant ID and returns a generated secret.',


  inputs: {
    entraTenantId: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: { description: '' },
    connectionAlreadyExists: {description: 'A Microsoft compliance tenant already exists for the provided entra tenant id.', statusCode: 409},
    missingOriginHeader: { description: 'No Origin header set', responseType: 'badRequest'},
  },


  fn: async function ({entraTenantId}) {

    // Look for an existing microsoftComplianceTenant record.
    let connectionAlreadyExists = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId});
    // If we found one with the provided tenant ID, return a 409 response.
    if(connectionAlreadyExists) {
      throw 'connectionAlreadyExists';
    }

    // Return a bad request response if the origin header is missing.
    if(!this.req.get('Origin')) {
      throw 'missingOriginHeader';
    }


    // Create a new database record for this tenant.
    let newTenant = await MicrosoftComplianceTenant.create({
      fleetServerSecret: sails.helpers.strings.random.with({len: 30}),
      entraTenantId: entraTenantId,
      fleetInstanceUrl: this.req.get('Origin'),
      setupCompleted: false,
    }).fetch();


    return {
      fleet_server_secret: newTenant.fleetServerSecret,// eslint-disable-line camelcase
      entra_tenant_id: entraTenantId,// eslint-disable-line camelcase
    };

  }


};

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
    connectionAlreadyExists: {description: 'A Microsoft compliance tenant already exists for the provided entra tenant id.'}
  },


  fn: async function ({entraTenantId, fleetLicenseKey}) {

    // Look for an existing microsoftComplianceTenant record.
    let connectionAlreadyExists = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId});
    // If we found one with the provided tenant ID, return an error.
    if(connectionAlreadyExists) {
      throw 'connectionAlreadyExists';
    }


    // Create a new database record for this tenant.
    let newTenant = await MicrosoftComplianceTenant.create({
      fleetServerSecret: sails.helpers.strings.random.with({len: 30}),
      entraTenantId: entraTenantId,
      licenseKey: fleetLicenseKey,
      fleetInstanceUrl: this.req.headers['Origin'],
      setupCompleted: false,
    });


    return {
      fleet_server_secret: newTenant.apiKey,
      entra_tenant_id: entraTenantId,
    };

  }


};

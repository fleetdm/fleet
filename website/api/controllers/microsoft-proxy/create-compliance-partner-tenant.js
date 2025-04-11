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
    success: { description: 'Details about a new Microsoft complaince tsenant have been returned to a Fleet isntance' },
    connectionAlreadyExists: {description: 'A Microsoft compliance tenant already exists for the provided entra tenant id.', statusCode: 409},
    missingOriginHeader: { description: 'No Origin header set', responseType: 'badRequest'},
  },


  fn: async function ({entraTenantId}) {

    // Return a badRequest response if the origin header is missing.
    if(!this.req.get('origin')) {// Note: req.get() is case insensitive.
      throw 'missingOriginHeader';
    }

    // Look for an existing microsoftComplianceTenant record using the requesting Fleet instances URL.
    let existingComplianceTenant = await MicrosoftComplianceTenant.findOne({fleetInstanceUrl: this.req.get('origin')});
    if(existingComplianceTenant) {
      // If we found one with the provided tenant ID, and setup was not completed, delete the incomplete compliance tenant and create a new one.
      if(!existingComplianceTenant.setupCompleted) {
        await MicrosoftComplianceTenant.destroyOne({id: existingComplianceTenant.id});
      } else {
        // If setup was already completed for the existing tenant, return a 409 response. (The user will need to delete the existing integration in the Fleet UI before creating a new one.)
        throw 'connectionAlreadyExists';
      }
    }

    // Create a new database record for this tenant.
    let newTenant = await MicrosoftComplianceTenant.create({
      fleetServerSecret: sails.helpers.strings.random.with({len: 30}),
      entraTenantId: entraTenantId,
      fleetInstanceUrl: this.req.get('origin'),
      setupCompleted: false,
    }).fetch();


    return {
      fleet_server_secret: newTenant.fleetServerSecret,// eslint-disable-line camelcase
      entra_tenant_id: entraTenantId,// eslint-disable-line camelcase
    };

  }


};

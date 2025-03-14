module.exports = {


  friendlyName: 'Create compliance partner tenant',


  description: '',


  inputs: {
    entra_tenant_id: {
      type: 'string',
      required: true,
    },
    fleet_license_key: {
      type: 'string',
      required: true,
    }
  },


  exits: {
    success: { description: ''},
    notACloudCustomer: { description: 'This request was not made by a managed cloud customer', responseType: 'badRequest' },
    connectionAlreadyExists: {description: 'A microsfot compliance tenant already exists for the provided entra tenant id.'}
  },


  fn: async function ({entra_tenant_id, fleet_license_key}) {

    // Return a bad request response if this request came from a non-managed cloud Fleet instance.
    if(!this.req.headers['Origin'] || !this.req.headers['Origin'].match(/cloud\.fleetdm\.com$/g)) {
      throw 'notACloudCustomer';
    }

    let connectionAlreadyExists = await MicrosoftComplianceTenant.findOne({entraTenantId: entra_tenant_id});
    if(connectionAlreadyExists) {
      throw 'connectionAlreadyExists';
    }

    // TODO: Do we need to validate the license key?

    // Create a new database record for this tenant.
    let newTenant = await MicrosoftComplianceTenant.create({
      apiKey: sails.helpers.strings.random.with({len: 30}),
      entraTenantId: entra_tenant_id,
      licenseKey: fleet_license_key,
      fleetInstanceUrl: this.req.headers['Origin'],
      setupCompleted: false,
    });


    return {
      fleet_server_secret: newTenant.apiKey,
      entra_tenant_id: entra_tenant_id,
      entra_application_id: sails.config.custom.entraApplicationId
    };

  }


};

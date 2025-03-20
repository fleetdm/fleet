module.exports = {


  friendlyName: 'Get compliance partner settings',


  description: '',

  inputs: {
    entraTenantId: {
      type: 'string',
      required: true,
    },
    fleetServerSecret: {
      type: 'string',
      requried: true,
    },
  },


  exits: {
    success: { description: 'The microsoft entra application ID was sent to a managed cloud instance.'},
  },


  fn: async function ({entraTenantId, fleetServerSecret}) {

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId, fleetServerSecret: fleetServerSecret});
    if(!informationAboutThisTenant) {
      return this.res.notFound();
    }
    if(informationAboutThisTenant.adminConsented) {
      return {
        entra_tenant_id: entraTenantId,
        setup_done: informationAboutThisTenant.setupCompleted,
        admin_consented: true
      };
    } else {
      // Generate a state token for the admin consent link.
      let stateTokenForThisAdminConsentLink = sails.helpers.strings.random.with({len: 30, style: 'url-friendly'});

      // Update the database record for this tenant to include the generated state token.
      await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({stateTokenForAdminConsent: stateTokenForThisAdminConsentLink});

      // Build an admin consent url for this request.
      let adminConsentUrlForThisTenant = `https://login.microsoftonline.com/${entraTenantId}/adminconsent?client_id=${encodeURIComponent(sails.config.custom.compliancePartnerClientId)}&state=${encodeURIComponent(stateTokenForThisAdminConsentLink)}&redirect_uri=${encodeURIComponent('fleetdm.com/api/v1/microsoft-compliance-partner/adminconsent')}`;

      return {
        entra_tenant_id: entraTenantId,
        setup_done: informationAboutThisTenant.setupCompleted,
        admin_consented: false,
        admin_consent_url: adminConsentUrlForThisTenant
      }
    }

  }


};

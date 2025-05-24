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
      required: true,
    },
  },


  exits: {
    success: { outputDescription: 'The setup and admin consent status of a Microsoft complianance tenant'},
  },


  fn: async function ({entraTenantId, fleetServerSecret}) {

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId, fleetServerSecret: fleetServerSecret});
    if(!informationAboutThisTenant) {
      return this.res.notFound();
    }

    // Otherwise, build an admin consent url for this tenant and include it in the response body.
    // Generate a state token for the admin consent link.
    let stateTokenForThisAdminConsentLink = sails.helpers.strings.random.with({len: 30, style: 'url-friendly'});
    // Update the database record for this tenant to include the generated state token.
    await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({stateTokenForAdminConsent: stateTokenForThisAdminConsentLink});
    // Build an admin consent url for this request.
    let adminConsentUrlForThisTenant = `https://login.microsoftonline.com/${entraTenantId}/adminconsent?client_id=${encodeURIComponent(sails.config.custom.compliancePartnerClientId)}&state=${encodeURIComponent(stateTokenForThisAdminConsentLink)}&redirect_uri=${encodeURIComponent(`${sails.config.custom.baseUrl}/api/v1/microsoft-compliance-partner/adminconsent`)}`;

    return {
      entra_tenant_id: entraTenantId,// eslint-disable-line camelcase
      setup_done: informationAboutThisTenant.setupCompleted,// eslint-disable-line camelcase
      admin_consented: informationAboutThisTenant.adminConsented,// eslint-disable-line camelcase
      admin_consent_url: adminConsentUrlForThisTenant,// eslint-disable-line camelcase
      setup_error: informationAboutThisTenant.setupError// eslint-disable-line camelcase
    };

  }


};

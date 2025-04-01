module.exports = {


  friendlyName: 'Get tenants admin consent status',


  description: 'Updates the admin consent status of a MicrosoftComplianceTenant record.',


  inputs: {
    tenant: {
      type: 'string',
      description: 'The entra tenant ID of a Microsoft Compliance Partner tenant'
    },
    state: {
      type: 'string',
      description: 'A token used to authenticate this request for a provided entra tenant.'
    },
    error: {
      type: 'string',
      description: 'This value is only present if an entra admin did approve the permissions for the compliance partner policy'
    },
    error_description: {// eslint-disable-line camelcase
      type: 'string',
      description: 'This value is only present if an entra admin did approve the permissions for the compliance partner policy'
    }
  },


  exits: {
    success: { description: 'The admin consent status of compliance tenant was successfully updated.', responseType: 'ok'},
    adminDidNotConsent: { description: 'An entra admin did not grant permissions to the Fleet compliance partner application for entra', responseType: 'ok'}
  },


  fn: async function ({tenant, state, error, error_description}) {// eslint-disable-line camelcase

    // If an error or error_description are provided, then the admin did not consent, and we will return a 200 response.
    if(error || error_description){// eslint-disable-line camelcase
      throw 'adminDidNotConsent';
    }

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: tenant, stateTokenForAdminConsent: state});
    // If no tenant was found that matches the provided tenant id and state, return a not found response.
    if(!informationAboutThisTenant) {
      return this.res.notFound();
    } else {
      // Otherwise, update the databse record for this tenant to show that the entra admin successfully consented, and clear our the state token from the record.
      await MicrosoftComplianceTenant.updateOne({id: informationAboutThisTenant.id}).set({adminConsented: true, stateTokenForAdminConsent: undefined});
    }


    return;
  }


};

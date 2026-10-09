module.exports = {


  friendlyName: 'Remove one compliance partner tenant',


  description: 'Updates a microsfot compliance tenant\'s status as "deprovisioned" and deletes the associated Database record',

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
    success: {
      description: 'The requesting entra tenant has been successfully deprovisioned.'
    },
    tenantNotFound: {
      description: 'A Microsoft compliance tenant could not be found using the provided information.',
      responseType: 'notFound',
    },
    microsoftApiRequestFailed: {description: 'An error occurred when sending a request to the Microsoft API.'},
    microsoftApiError: {description: 'The Microsoft API returned an unexpected response.'},
  },


  fn: async function ({entraTenantId, fleetServerSecret}) {

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({entraTenantId: entraTenantId, fleetServerSecret: fleetServerSecret});
    if(!informationAboutThisTenant) {
      throw 'tenantNotFound';
    }

    // If setup was completed, we will need to deprovision this Complaince tenant, otherwise, we will only delete the databse record.
    if(informationAboutThisTenant.setupCompleted){

      let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
        complianceTenantRecordId: informationAboutThisTenant.id
      })
      .intercept('microsoftApiRequestFailed', 'microsoftApiRequestFailed')
      .intercept('microsoftApiError', 'microsoftApiError');

      let accessToken = tokenAndApiUrls.manageApiAccessToken;
      let tenantDataSyncUrl = tokenAndApiUrls.tenantDataSyncUrl;


      // Deprovison this tenant
      let deprovisionTenantResponse = await sails.helpers.http.sendHttpRequest.with({
        method: 'PUT',
        url: `${tenantDataSyncUrl}/PartnerTenants(guid'${informationAboutThisTenant.entraTenantId}')?api-version=1.6`,
        headers: {
          'Authorization': `Bearer ${accessToken}`
        },
        body: {
          Provisioned: 2,// 1 = provisioned, 2 = deprovisioned.
          PartnerEnrollmentUrl: `https://fleetdm.com/microsoft-compliance-partner/enroll`,
          PartnerRemediationUrl: `https://fleetdm.com/microsoft-compliance-partner/remediate`,
        }
      }).intercept({raw: {statusCode: 401}}, async (err)=>{
        // If the Microsoft API rejected the cached access token, clear this tenant's cached tokens and URLs to force re-authentication on the next request.
        await sails.helpers.microsoftProxy.clearCacheForTenant.with({entraTenantId});
        sails.log.warn(`When deprovisioning a Microsoft compliance tenant, the cached access token was rejected. Full error: ${require('util').inspect(err, {depth: 3})}`);
        return 'microsoftApiError';
      }).intercept((err)=>{
        return new Error({error: `an error occurred when deprovisioning a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`});
      });
      // Log responses from Micrsoft APIs for Fleet's integration
      if(informationAboutThisTenant.fleetInstanceUrl === 'https://dogfood.fleetdm.com') {
        sails.log.info(`Microsoft proxy: remove-one-compliance-partner-tenant deprovisioned a tenant: ${deprovisionTenantResponse.body}`);
      }
    }

    await MicrosoftComplianceTenant.destroyOne({id: informationAboutThisTenant.id});

    // Remove any cached access tokens and data sync URLs for the removed tenant.
    await sails.helpers.microsoftProxy.clearCacheForTenant.with({entraTenantId});


    // All done.
    return this.res.json({});

  }


};

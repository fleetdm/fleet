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
      requried: true,
    },
  },


  exits: {
    success: {
      description: 'The requesting entra tenant has been successfully deprovisioned.'
    },
    tenantNotFound: {
      description: 'A Microsoft compliance tenant could not be found using the provided information.',
      responseType: 'notFound',
    }
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
      });

      let accessToken = tokenAndApiUrls.manageApiAccessToken;
      let tenantDataSyncUrl = tokenAndApiUrls.tenantDataSyncUrl;


      // Deprovison this tenant
      await sails.helpers.http.sendHttpRequest.with({
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
      }).intercept((err)=>{
        return new Error({error: `an error occurred when deprovisioning a Microsoft compliance tenant. Full error: ${require('util').inspect(err, {depth: 3})}`});
      });
    }

    await MicrosoftComplianceTenant.destroyOne({id: informationAboutThisTenant.id});


    // All done.
    return this.res.json({});

  }


};

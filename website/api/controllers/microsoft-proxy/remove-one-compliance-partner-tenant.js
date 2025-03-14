module.exports = {


  friendlyName: 'Remove one compliance partner tenant',


  description: '',


  exits: {

  },


  fn: async function () {

    // Return a bad request response if this request came from a non-managed cloud Fleet instance.
    if(!this.req.headers['Origin'] || !this.req.headers['Origin'].match(/cloud\.fleetdm\.com$/g)) {
      throw 'notACloudCustomer';
    }

    if(!this.req.headers['Authorization']) {
      return this.res.unauthorized();
    }

    let tokenForThisRequest = authHeaderValue.split('Bearer ')[1];
    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({apiKey: tokenForThisRequest});
    if(!informationAboutThisTenant) {
      return new Error({error: 'No MicrosoftComplianceTenant record was found that matches the provided API key.'});// TODO: return a more clear error.
    }

    let tokenAndApiUrls = await sails.helpers.microsoftProxy.getAccessTokenAndApiUrls.with({
      complianceTenantRecordId: informationAboutThisTenant.id
    });

    let accessToken = tokenAndApiUrls.accessToken;
    let tenantDataSyncUrl = tokenAndApiUrls.tenantDataSyncUrl;


    // Deprovison this tenant
    await sails.helpers.http.sendHtttpRequest.with({
      method: 'PUT',
      url: `${tenantDataSyncUrl}/${encodeURIComponent(`PartnerTenants(guid${informationAboutThisTenant.entraTenantId}`)}?api-version=1.0`,
      headers: {
        'Authorization': `Bearer ${accessToken}`
      },
      body: {
        Provisioned: 2,// 1 = provisioned, 2 = deprovisioned.
        PartnerEnrollmentUrl: '', //TODO: how do we get this, the example in microsoft's docs are using customer.com/enrollment, so does this need to be a value of a url on the connected Fleet instance?
        PartnerRemediationUrl: '', // TODO: same as the above.
      }
    });

    await MicrosoftComplianceTenant.destroyOne({id: informationAboutThisTenant.id});


    // All done.
    return;

  }


};

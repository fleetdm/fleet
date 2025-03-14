module.exports = {


  friendlyName: 'Setup compliance partner tenant',


  description: '',


  inputs: {

  },


  exits: {
    notACloudCustomer: { description: 'This request was not made by a managed cloud customer', responseType: 'badRequest' },
  },


  fn: async function () {

    // Return a bad request response if this request came from a non-managed cloud Fleet instance.
    if(!this.req.headers['Origin'] || !this.req.headers['Origin'].match(/cloud\.fleetdm\.com$/g)) {
      throw 'notACloudCustomer';
    }

    if(!this.req.headers['Authentication']) {
      return this.res.unauthorized();
    }

    let authHeaderValue = this.req.headers['Authentication']
    let tokenForThisRequest = authHeaderValue.split('Bearer ')[1];
    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({apiKey: tokenForThisRequest});
    if(!informationAboutThisTenant) {
      return new Error({error: 'No MicrosoftComplianceTenant record was found that matches the provided API key.'})// TODO: return a more clear error.
    }
    // Get the compliance partner API urls for this

    // [?]: https://learn.microsoft.com/en-us/graph/api/resources/serviceprincipal
    let servicePrincipalResponse = await sails.helpers.http.get(`https://graph.microsoft.com/v1.0/servicePrincipals?$filter=${encodeURIComponent(`appId eq ${informationAboutThisTenant.entraTenantId}`)}`);

    // TODO: verify that the servicePrincipal object is not nested in the response from the Microsoft graph API.
    let servicePrincipalObjectId = servicePrincipalResponse.id;

    // [?]: https://learn.microsoft.com/en-us/graph/api/group-list-endpoints
    let servicePrincipalEndpointResponse = await sails.helpers.http.get(`https://graph.microsoft.com/v1.0/servicePrincipals/${servicePrincipalObjectId}/endpoints`);

    let endpointsInResponse = servicePrincipalEndpointResponse.value;


    let tenantDataSyncUrl = _.find(endpointsInResponse, {providerName: 'PartnerTenantDataSyncService'}).uri;

    // provision the new tenant.



    // return a 200 response if everything was setup correctly
    return;

  }


};

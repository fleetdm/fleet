module.exports = {


  friendlyName: 'Get access token and api urls',


  description: 'Retreives an access token and the URLS of API endpoints for a Microsoft compliance tenant',


  extendedDescription: 'Access tokens and data sync URLs are cached per-tenant, so most calls of this helper make no requests to Microsoft. Access tokens are cached until five minutes before they expire, and data sync URLs are cached for 24 hours. Use the clearCacheForTenant helper to force re-authentication and URL re-discovery on the next call.',


  inputs: {
    complianceTenantRecordId: {
      type: 'number',
      required: true,
    }
  },


  exits: {

    success: {
      outputFriendlyName: 'Access token and api urls',
    },

    microsoftApiRequestFailed: {
      description: 'A request to a Microsoft API failed to send.',
    },

    microsoftApiError: {
      description: 'A Microsoft API returned an unexpected response.',
    },

  },


  fn: async function ({complianceTenantRecordId}) {

    let informationAboutThisTenant = await MicrosoftComplianceTenant.findOne({id: complianceTenantRecordId});
    if(!informationAboutThisTenant) {
      throw new Error(`No matching tenant record could be found with the specified ID. (${complianceTenantRecordId})`);
    }

    // Note: these cache keys are shared with the clearCacheForTenant helper.
    let accessTokensCacheKey = `microsoft-proxy:access-tokens:${informationAboutThisTenant.entraTenantId}`;
    let apiUrlsCacheKey = `microsoft-proxy:api-urls:${informationAboutThisTenant.entraTenantId}`;

    // Sends a request to a Microsoft API, retrying once after a short pause if the request fails
    // with a transient error: a network error, a 401 response (Microsoft intermittently rejects
    // freshly-issued access tokens), or a 5xx response. Non-transient and persistent failures are
    // mapped to this helper's microsoftApiRequestFailed/microsoftApiError exits.
    let sendMicrosoftApiRequestWithRetry = async (getRequestDeferred, requestDescription)=>{
      let isTransientError = (err)=>{
        return err.code === 'requestFailed' || Boolean(err.raw && _.contains([401, 500, 502, 503, 504], err.raw.statusCode));
      };
      let requestError;
      try {
        return await getRequestDeferred();
      } catch(err) {
        if(isTransientError(err)) {
          await sails.helpers.flow.pause(1000);
          try {
            return await getRequestDeferred();
          } catch(retryErr) {
            requestError = retryErr;
          }
        } else {
          requestError = err;
        }
      }
      if(requestError.raw && requestError.raw.statusCode === 401) {
        // If Microsoft rejected the access token used for this request, clear this tenant's cached tokens to force re-authentication on the next request.
        await sails.helpers.cache.destroy.with({key: accessTokensCacheKey});
      }
      if(requestError.code === 'requestFailed') {
        throw 'microsoftApiRequestFailed';
      }
      sails.log.warn(`When ${requestDescription} for a Microsoft compliance tenant (entra tenant id: ${informationAboutThisTenant.entraTenantId}), an error occurred. Full error: ${require('util').inspect(requestError, {depth: 3})}`);
      throw 'microsoftApiError';
    };

    let graphAccessToken;
    let manageApiAccessToken;
    let cachedAccessTokens = await sails.helpers.cache.get.with({key: accessTokensCacheKey});
    if(cachedAccessTokens && cachedAccessTokens.graphAccessToken && cachedAccessTokens.manageApiAccessToken) {
      graphAccessToken = cachedAccessTokens.graphAccessToken;
      manageApiAccessToken = cachedAccessTokens.manageApiAccessToken;
    } else {
      // Get a graph access token for this tenant
      let graphAccessTokenResponse = await sendMicrosoftApiRequestWithRetry(()=>{
        return sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          url: `https://login.microsoftonline.com/${informationAboutThisTenant.entraTenantId}/oauth2/v2.0/token`,
          enctype: 'application/x-www-form-urlencoded',
          body: {
            client_id: sails.config.custom.compliancePartnerClientId,// eslint-disable-line camelcase
            scope: 'https://graph.microsoft.com/.default',
            client_secret: sails.config.custom.compliancePartnerClientSecret,// eslint-disable-line camelcase
            grant_type: 'client_credentials'// eslint-disable-line camelcase
          }
        });
      }, 'sending a request to get a graph access token');
      // Get a management API access token for this tenant
      let manageAccessTokenResponse = await sendMicrosoftApiRequestWithRetry(()=>{
        return sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          url: `https://login.microsoftonline.com/${informationAboutThisTenant.entraTenantId}/oauth2/v2.0/token`,
          enctype: 'application/x-www-form-urlencoded',
          body: {
            client_id: sails.config.custom.compliancePartnerClientId,// eslint-disable-line camelcase
            scope: 'https://api.manage.microsoft.com//.default',
            client_secret: sails.config.custom.compliancePartnerClientSecret,// eslint-disable-line camelcase
            grant_type: 'client_credentials'// eslint-disable-line camelcase
          }
        });
      }, 'sending a request to get a management API access token');

      // Parse the json response bodies to get the access tokens and their expiries.
      let secondsUntilTokensExpire;
      try {
        let parsedGraphTokenResponse = JSON.parse(graphAccessTokenResponse.body);
        let parsedManageTokenResponse = JSON.parse(manageAccessTokenResponse.body);
        graphAccessToken = parsedGraphTokenResponse.access_token;
        manageApiAccessToken = parsedManageTokenResponse.access_token;
        secondsUntilTokensExpire = Math.min(Number(parsedGraphTokenResponse.expires_in), Number(parsedManageTokenResponse.expires_in));
      } catch(err){
        throw new Error(`When sending a request to get an access token for a Microsoft compliance tenant, an error occured. full error: ${require('util').inspect(err)}`);
      }
      if(!graphAccessToken || !manageApiAccessToken) {
        sails.log.warn(`When getting access tokens for a Microsoft compliance tenant (entra tenant id: ${informationAboutThisTenant.entraTenantId}), a response from Microsoft did not include an access token.`);
        throw 'microsoftApiError';
      }

      // Cache the tokens until five minutes before they expire, so a request never uses a token that expires while it is in flight.
      let tokenCacheTtlInSeconds = secondsUntilTokensExpire - 60 * 5;
      if(tokenCacheTtlInSeconds > 0) {
        await sails.helpers.cache.set.with({
          key: accessTokensCacheKey,
          value: {
            graphAccessToken: graphAccessToken,
            manageApiAccessToken: manageApiAccessToken,
          },
          ttl: tokenCacheTtlInSeconds,
        });
      }
    }

    let tenantDataSyncUrl;
    let deviceDataSyncUrl;
    let cachedApiUrls = await sails.helpers.cache.get.with({key: apiUrlsCacheKey});
    if(cachedApiUrls && cachedApiUrls.tenantDataSyncUrl && cachedApiUrls.deviceDataSyncUrl) {
      tenantDataSyncUrl = cachedApiUrls.tenantDataSyncUrl;
      deviceDataSyncUrl = cachedApiUrls.deviceDataSyncUrl;
    } else {
      // [?]: https://learn.microsoft.com/en-us/graph/api/resources/serviceprincipal
      let servicePrincipalResponse = await sendMicrosoftApiRequestWithRetry(()=>{
        return sails.helpers.http.get.with({
          url: `https://graph.microsoft.com/v1.0/servicePrincipals?$filter=${encodeURIComponent(`appId eq '0000000a-0000-0000-c000-000000000000'`)}`,
          headers: {
            'Authorization': `Bearer ${graphAccessToken}`
          }
        });
      }, 'sending a request to get the Intune service principal');

      if(!servicePrincipalResponse.value || !servicePrincipalResponse.value[0]) {
        sails.log.warn(`When sending a request to get the Intune service principal of a Microsoft compliance tenant (entra tenant id: ${informationAboutThisTenant.entraTenantId}), the response from Microsoft did not include a service principal.`);
        throw 'microsoftApiError';
      }
      let servicePrincipalObjectId = servicePrincipalResponse.value[0].id;

      // [?]: https://learn.microsoft.com/en-us/graph/api/group-list-endpoints
      let servicePrincipalEndpointResponse = await sendMicrosoftApiRequestWithRetry(()=>{
        return sails.helpers.http.get.with({
          url: `https://graph.microsoft.com/v1.0/servicePrincipals/${servicePrincipalObjectId}/endPoints`,
          headers: {
            'Authorization': `Bearer ${graphAccessToken}`
          },
        });
      }, 'sending a request to get the endpoints of the Intune service principal');

      let endpointsInResponse = servicePrincipalEndpointResponse.value;

      let tenantDataSyncService = _.find(endpointsInResponse, {providerName: 'PartnerTenantDataSyncService'});
      if(!tenantDataSyncService) {
        throw new Error(`When sending a request to get the PartnerTenantDataSyncService service principal of a Microsoft compliance tenant, no PartnerTenantDataSyncService service principal was found.`);
      }
      tenantDataSyncUrl = tenantDataSyncService.uri;

      let deviceDataSyncService = _.find(endpointsInResponse, {providerName: 'PartnerDeviceDataSyncService'});
      if(!deviceDataSyncService) {
        throw new Error(`When sending a request to get the PartnerDeviceDataSyncService service principal of a Microsoft compliance tenant, no PartnerDeviceDataSyncService service principal was found.`);
      }
      deviceDataSyncUrl = deviceDataSyncService.uri;

      // Data sync URLs are effectively static per tenant, so cache them for 24 hours.
      await sails.helpers.cache.set.with({
        key: apiUrlsCacheKey,
        value: {
          tenantDataSyncUrl: tenantDataSyncUrl,
          deviceDataSyncUrl: deviceDataSyncUrl,
        },
        ttl: 24 * 60 * 60,
      });
    }

    return {
      manageApiAccessToken,
      graphAccessToken,
      tenantDataSyncUrl,
      deviceDataSyncUrl,
    };
  }


};

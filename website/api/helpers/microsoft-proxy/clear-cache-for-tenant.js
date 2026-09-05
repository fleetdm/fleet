module.exports = {


  friendlyName: 'Clear cache for tenant',


  description: 'Removes the cached Microsoft access tokens and data sync URLs of a compliance tenant.',


  extendedDescription: 'Used to force re-authentication and data sync URL re-discovery on the next call of the getAccessTokenAndApiUrls helper for a tenant, e.g. after a request using a cached token or URL fails.',


  sideEffects: 'idempotent',


  inputs: {
    entraTenantId: {
      type: 'string',
      required: true,
    },
  },


  exits: {

    success: {
      description: 'Any cached access tokens and data sync URLs of the specified tenant were removed.',
    },

  },


  fn: async function ({entraTenantId}) {

    // Note: these cache keys are shared with the getAccessTokenAndApiUrls helper.
    await sails.helpers.cache.destroyCachedValue.with({key: `microsoft-proxy:access-tokens:${entraTenantId}`});
    await sails.helpers.cache.destroyCachedValue.with({key: `microsoft-proxy:api-urls:${entraTenantId}`});

  }


};

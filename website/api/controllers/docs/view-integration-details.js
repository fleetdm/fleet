module.exports = {


  friendlyName: 'View integration details',


  description: 'Display "Integration details" page.',


  inputs: {
    categorySlug: {
      type: 'string',
      required: true,
      description: 'The first path segment after /integrate/ (e.g. "idp").',
      example: 'idp'
    },
    integrationSlug: {
      type: 'string',
      required: true,
      description: 'The slug of the integration (e.g. "okta").',
      example: 'okta'
    },
  },


  exits: {

    success: {
      viewTemplatePath: 'pages/docs/integration-details'
    },


    badConfig: {
      responseType: 'badConfig'
    },

    notFound: {
      responseType: 'notFound'
    },

  },


  fn: async function ({categorySlug, integrationSlug}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.integrations)) {
      throw {badConfig: 'builtStaticContent.integrations'};
    }

    let requestedPath = '/integrate/' + categorySlug + '/' + integrationSlug;
    let thisIntegration = _.find(sails.config.builtStaticContent.integrations, { url: requestedPath });
    if (!thisIntegration) {
      throw 'notFound';
    }

    let thisCategory = _.find(sails.config.builtStaticContent.integrationCategories, { slug: thisIntegration.category });

    // Respond with view.
    return {
      thisIntegration,
      thisCategory,
      integrationsYmlRepoPath: sails.config.builtStaticContent.integrationsYmlRepoPath,
      pageTitleForMeta: thisIntegration.name + ' integration',
      pageDescriptionForMeta: thisIntegration.description,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

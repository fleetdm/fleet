module.exports = {


  friendlyName: 'View integrations directory',


  description: 'Display "Integrations directory" page, either for all integrations or for a single category.',


  inputs: {
    categorySlug: {
      type: 'string',
      description: 'The slug of an integration category (e.g. "identity-provider"). If omitted, all integrations are shown.',
      example: 'identity-provider'
    },
  },


  exits: {

    success: {
      viewTemplatePath: 'pages/docs/integrations-directory'
    },

    redirect: {
      responseType: 'redirect',
      description: 'The requested URL is an alias for a category or integration page.'
    },

    badConfig: {
      responseType: 'badConfig'
    },

    notFound: {
      responseType: 'notFound'
    },

  },


  fn: async function ({categorySlug}) {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.integrations) || !_.isArray(sails.config.builtStaticContent.integrationCategories)) {
      throw {badConfig: 'builtStaticContent.integrations'};
    }

    let allIntegrations = sails.config.builtStaticContent.integrations;
    let allCategories = sails.config.builtStaticContent.integrationCategories;
    let buildingBlocks = sails.config.builtStaticContent.integrationBuildingBlocks || [];

    let selectedCategory;
    let isBuildAnythingPage = categorySlug === 'build-anything';
    if (isBuildAnythingPage) {
      selectedCategory = { name: 'Build anything', slug: 'build-anything', url: '/integrate/build-anything' };
    } else if (categorySlug) {
      let requestedPath = '/integrate/' + categorySlug;
      selectedCategory = _.find(allCategories, { slug: categorySlug });
      if (!selectedCategory) {
        // A single-segment URL can also be an alias for a category or an integration (e.g. /integrate/idp, /integrate/sentinel-one).
        let aliasTarget = _.find(allCategories.concat(allIntegrations), (categoryOrIntegration)=>{
          return categoryOrIntegration.redirectsFrom.includes(requestedPath);
        });
        if (aliasTarget) {
          throw {redirect: aliasTarget.url};
        }
        throw 'notFound';
      }
    }

    // builtStaticContent.integrations is already sorted by name.
    let integrationsToShow = allIntegrations;
    if (isBuildAnythingPage) {
      integrationsToShow = [];
    } else if (selectedCategory) {
      integrationsToShow = _.filter(allIntegrations, (integration)=>{
        return integration.category === selectedCategory.slug || integration.alsoListedIn.includes(selectedCategory.slug);
      });
    }

    let pageTitleForMeta = isBuildAnythingPage ? 'Build anything with Fleet' : selectedCategory ? selectedCategory.name + ' integrations' : 'Integrations';

    // Respond with view.
    return {
      integrationsToShow,
      buildingBlocks,
      isBuildAnythingPage,
      integrationsYmlRepoPath: sails.config.builtStaticContent.integrationsYmlRepoPath,
      allCategories: _.filter(allCategories, (category)=>{
        return _.any(allIntegrations, (integration)=>{
          return integration.category === category.slug || integration.alsoListedIn.includes(category.slug);
        });
      }),
      selectedCategory,
      numberOfIntegrations: allIntegrations.length,
      pageTitleForMeta,
      algoliaPublicKey: sails.config.custom.algoliaPublicKey,
    };

  }


};

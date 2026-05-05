module.exports = {


  friendlyName: 'View basic whitepaper',


  description: 'Display "Basic whitepaper" page.',

  inputs: {
    slug: {
      type: 'string',
      description: 'The slug of the whitepaper article that will be displayed to the user',
      required: true,
    }
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/articles/basic-whitepaper'
    },

    badConfig: {
      responseType: 'badConfig'
    },

    notFound: {
      responseType: 'notFound'
    },


  },


  fn: async function ({slug}) {


    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.markdownPages) || !sails.config.builtStaticContent.markdownPages) {
      throw {badConfig: 'builtStaticContent.markdownPages'};
    }

    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, { url: '/whitepapers/'+encodeURIComponent(slug) });
    if (!thisPage) {
      throw 'notFound';
    }

    let pageTitleForMeta;
    let pageDescriptionForMeta;

    if(thisPage.meta.articleTitle) {
      pageTitleForMeta = thisPage.meta.articleTitle;
    }

    if(thisPage.meta.description) {
      pageDescriptionForMeta = thisPage.meta.description;
    }

    // Respond with view.
    return {
      pageTitleForMeta,
      pageDescriptionForMeta,
      path: require('path'),
      thisPage: thisPage,
    };

  }


};

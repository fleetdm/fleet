module.exports = {


  friendlyName: 'View basic comparison',


  description: 'Display "Basic comparison" page.',

  inputs: {
    slug: {
      type: 'string',
      description: 'The slug of the comparison page that will be displayed to the user',
      required: true,
    }
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/basic-comparison'
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

    let thisPage = _.find(sails.config.builtStaticContent.markdownPages, { url: '/compare/'+encodeURIComponent(slug) });
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

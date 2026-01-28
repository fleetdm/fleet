module.exports = {


  friendlyName: 'View fleet gitops',


  description: 'Display "Fleet gitops" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/fleet-gitops'
    },

    badConfig: {
      responseType: 'badConfig'
    },

  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);


    // Respond with view.
    return {
      testimonialsForScrollableTweets
    };

  }


};

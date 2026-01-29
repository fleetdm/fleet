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
    let testimonialsForScrollableTweets = _.uniq(_.clone(sails.config.builtStaticContent.testimonials), (quote)=>{
      return quote.quoteAuthorName.toLowerCase();
    });

    // Respond with view.
    return {
      testimonialsForScrollableTweets
    };

  }


};

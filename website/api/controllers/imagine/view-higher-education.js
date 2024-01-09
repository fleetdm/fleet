module.exports = {


  friendlyName: 'View higher education',


  description: 'Display "Higher education" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/imagine/higher-education'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = sails.config.builtStaticContent.testimonials;
    // Respond with view.
    return {
      testimonialsForScrollableTweets,
    };

  }


};

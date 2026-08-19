module.exports = {


  friendlyName: 'View infrastructure-as-code',


  description: 'Display "Fleet infrastructure-as-code" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/infrastructure-as-code'
    },

    badConfig: {
      responseType: 'badConfig'
    },

  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials)) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Only filter testimonials when static content has been built.
    // If the build-static-content script was not run, we'll show a placeholder testimonial that is added by the custom hook.
    if (sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      testimonialsForScrollableTweets = _.uniq(testimonialsForScrollableTweets, (quote)=>{
        return quote.quoteAuthorName.toLowerCase();
      });
    }//ﬁ

    // Respond with view.
    return {
      testimonialsForScrollableTweets
    };

  }


};

module.exports = {


  friendlyName: 'View visibility and reporting',


  description: 'Display "Visibility and reporting" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/visibility-and-reporting'
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

    // Only filter and sort testimonials when static content has been built.
    // If the build-static-content script was not run, we'll show a placeholder testimonial that is added by the custom hook.
    if(sails.config.builtStaticContent.compiledPagePartialsAppPath) {

      // Specify an order for the testimonials on this page using the last names of quote authors
      let testimonialOrderForThisPage = [
        'Scott MacVicar',
        'Luis Madrigal',
        'Nick Fohs',
        'Kenny Botelho',
        'Charles Zaffery',
        'Arsenio Figueroa',
        'Matt Carr',
        'Andre Shields',
        'Erik Gomez',
      ];
      // Filter the testimonials by product category and the filtered list we built above.
      testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
        return _.contains(testimonial.productCategories, 'Observability') && _.contains(testimonialOrderForThisPage, testimonial.quoteAuthorName);
      });

      testimonialsForScrollableTweets.sort((a, b)=>{
        if(testimonialOrderForThisPage.indexOf(a.quoteAuthorName) === -1){
          return 1;
        } else if(testimonialOrderForThisPage.indexOf(b.quoteAuthorName) === -1) {
          return -1;
        }
        return testimonialOrderForThisPage.indexOf(a.quoteAuthorName) - testimonialOrderForThisPage.indexOf(b.quoteAuthorName);
      });
    }

    // Respond with view.
    return {
      testimonialsForScrollableTweets,
    };

  }


};

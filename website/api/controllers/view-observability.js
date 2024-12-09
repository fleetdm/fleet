module.exports = {


  friendlyName: 'View observability',


  description: 'Display "Observability" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/observability'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);
    // Default the pagePersonalization to the user's primaryBuyingSituation if it is set, otherwise, default to the eo-it view..
    let pagePersonalization = this.req.session.primaryBuyingSituation ? this.req.session.primaryBuyingSituation : 'eo-it';
    // If a purpose query parameter is set, update the pagePersonalization value.
    // Note: This is the only page we're using this method instead of using the primaryBuyingSiutation value set in the users session.
    // This lets us link to the security and IT versions of the endpoint ops page from the unpersonalized homepage without changing the users primaryBuyingSituation.
    if(this.req.param('purpose') === 'it'){
      pagePersonalization = 'eo-it';
    } else if(this.req.param('purpose') === 'security'){
      pagePersonalization = 'eo-security';
    }

    // Specify an order for the testimonials on this page using the last names of quote authors
    let testimonialOrderForThisPage = [
      'Eric Tan',
      'Kenny Botelho',
      'Ahmed Elshaer',
      'Arsenio Figueroa',
      'Brendan Shaklovitz',
      'Andre Shields',
      'Scott MacVicar',
      'Erik Gomez',
      'Mike Arpaia',
      'Chandra Majumdar',
      'Charles Zaffery',
      'Tom Larkin',
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

    // Respond with view.
    return {
      testimonialsForScrollableTweets,
      pagePersonalization,
    };

  }


};

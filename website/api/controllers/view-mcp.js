module.exports = {


  friendlyName: 'View mcp',


  description: 'Display "Mcp" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/mcp'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials)) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrollable-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Only filter and sort testimonials when static content has been built.
    // If the build-static-content script was not run, we'll show a placeholder testimonial that is added by the custom hook.
    if (sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      // Specify an order for the testimonials on this page using the names of quote authors.
      // Fleet MCP is built on osquery and aimed at security and observability teams, so we
      // lead with security, incident response, and osquery voices.
      let testimonialOrderForThisPage = [
        'Mike Arpaia',
        'Nico Waisman',
        'Dan Grzelak',
        'Andy Gombar',
        'Ahmed Elshaer',
        'Andre Shields',
        'Eric Tan',
        'Arsenio Figueroa',
        'Austin Anderson',
        'Joe Pistone',
        'Luis Madrigal',
      ];

      // Filter the testimonials by product category and the curated author list above.
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
    }//ﬁ

    // Respond with view.
    return {
      testimonialsForScrollableTweets,
    };

  }


};

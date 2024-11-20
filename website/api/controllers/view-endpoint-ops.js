module.exports = {


  friendlyName: 'View endpoint ops',


  description: 'Display "Endpoint ops" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/endpoint-ops'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);
    // Default the pagePersonalization to the user's primaryBuyingSituation.
    let pagePersonalization = this.req.session.primaryBuyingSituation;
    // If a purpose query parameter is set, update the pagePersonalization value.
    // Note: This is the only page we're using this method instead of using the primaryBuyingSiutation value set in the users session.
    // This lets us link to the security and IT versions of the endpoint ops page from the unpersonalized homepage without changing the users primaryBuyingSituation.
    if(this.req.param('purpose') === 'security'){
      pagePersonalization = 'eo-security';
    } else {
      // Default to IT personalization if no primaryBuyingSituation or purpose query parameter is set.
      pagePersonalization = 'eo-it';
    }


    // Specify an order for the testimonials on this page using the last names of quote authors
    let testimonialOrderForThisPage = ['Charles Zaffery','Dan Grzelak','Nico Waisman','Tom Larkin','Austin Anderson','Erik Gomez','Nick Fohs','Brendan Shaklovitz','Mike Arpaia','Andre Shields','Dhruv Majumdar','Ahmed Elshaer','Abubakar Yousafzai','Wes Whetstone','Kenny Botelho', 'Chandra Majumdar','Eric Tan', 'Alvaro Gutierrez', 'Joe Pistone'];
    if(['eo-it', 'mdm'].includes(pagePersonalization)){
      testimonialOrderForThisPage = [ 'Eric Tan','Erik Gomez', 'Tom Larkin', 'Nick Fohs', 'Wes Whetstone', 'Mike Arpaia', 'Kenny Botelho', 'Alvaro Gutierrez'];
    } else if(['eo-security', 'vm'].includes(pagePersonalization)){
      testimonialOrderForThisPage = ['Nico Waisman','Charles Zaffery','Abubakar Yousafzai','Eric Tan','Mike Arpaia','Chandra Majumdar','Ahmed Elshaer','Brendan Shaklovitz','Austin Anderson','Dan Grzelak','Dhruv Majumdar','Alvaro Gutierrez', 'Joe Pistone'];
    }
    // Filter the testimonials by product category and the filtered list we built above.
    testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Endpoint operations') && _.contains(testimonialOrderForThisPage, testimonial.quoteAuthorName);
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

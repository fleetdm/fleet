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

    // Filter the testimonials by product category
    testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Endpoint operations');
    });

    // Specify an order for the testimonials on this page using the last names of quote authors
    let testimonialOrderForThisPage = ['Zaffery','Grzelak','Waisman','Larkin','Anderson','Gomez','Fohs','Shaklovitz','Arpaia','Shields','Majumdar','Elshaer','Yousafzai','Ravazzolo','Whetstone','Botelho'];
    testimonialsForScrollableTweets.sort((a, b)=>{
      if(testimonialOrderForThisPage.indexOf(a.quoteAuthorName.split(' ')[1]) === -1){
        return 1;
      } else if(testimonialOrderForThisPage.indexOf(b.quoteAuthorName.split(' ')[1]) === -1) {
        return -1;
      }
      return testimonialOrderForThisPage.indexOf(a.quoteAuthorName.split(' ')[1]) - testimonialOrderForThisPage.indexOf(b.quoteAuthorName.split(' ')[1]);
    });

    // Respond with view.
    return {
      testimonialsForScrollableTweets,
    };

  }


};

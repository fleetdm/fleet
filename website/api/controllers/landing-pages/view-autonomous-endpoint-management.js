module.exports = {


  friendlyName: 'View autonomous endpoint management',


  description: 'Display "Autonomous endpoint management" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/autonomous-endpoint-management'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    let testimonialOrderForThisPage = [
      'Dan Jackson',
      'Wes Whetstone',
      'Mike Meyer',
      'Eric Tan',
      'Erik Gomez',
      'Nick Fohs',
      'Dan Grzelak',
      'Austin Anderson',
      'Charles Zaffery',
      'Andre Shields',
      'Nico Waisman',
    ];

    testimonialsForScrollableTweets = _.filter(testimonialsForScrollableTweets, (testimonial)=>{
      return _.contains(testimonial.productCategories, 'Software management') || _.contains(testimonial.productCategories, 'Device management');
    });

    testimonialsForScrollableTweets.sort((a, b)=>{
      if(testimonialOrderForThisPage.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialOrderForThisPage.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialOrderForThisPage.indexOf(a.quoteAuthorName) - testimonialOrderForThisPage.indexOf(b.quoteAuthorName);
    });

    return {
      testimonialsForScrollableTweets
    };

  }


};

module.exports = {


  friendlyName: 'View open source',


  description: 'Display "Open source" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/open-source'
    },
    badConfig: { responseType: 'badConfig' },
  },


  fn: async function () {
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the <scrollable-tweets> component.
    let testimonialsForScrollableTweets = _.clone(sails.config.builtStaticContent.testimonials);

    // Order testimonials to surface ones that speak to open source, transparency, and community.
    let testimonialOrderForThisPage = [
      'nico waisman',
      'Scott MacVicar',
      'John O\'Nolan',
      'Brendan Shaklovitz',
      'Mike Arpaia',
      'Justin LaBo',
      'Bart Reardon',
      'Wes Whetstone',
      'Roger Cantrell',
      'u/Heteronymous',
    ];

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
    };

  }


};

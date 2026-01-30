module.exports = {


  friendlyName: 'View testimonials',


  description: 'Display "Testimonials" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/testimonials'
    }

  },


  fn: async function () {

    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.testimonials) || !sails.config.builtStaticContent.compiledPagePartialsAppPath) {
      throw {badConfig: 'builtStaticContent.testimonials'};
    }
    // Get testimonials for the page contents
    let testimonials = _.clone(sails.config.builtStaticContent.testimonials);


    let testimonialAuthorsToShowOnThisPage = [
      'Scott MacVicar',
      'Nick Fohs',
      'Wes Whetstone',
      'Erik Gomez',
      'matt carr',
      'nico waisman',
      'Kenny Botelho',
      'Dan Grzelak',
      'Eric Tan',
    ];

    let filteredTestimonialsForThisPage = _.filter(testimonials, (testimonial)=>{
      return testimonialAuthorsToShowOnThisPage.includes(testimonial.quoteAuthorName);
    });
    filteredTestimonialsForThisPage.sort((a, b)=>{
      if(testimonialAuthorsToShowOnThisPage.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(testimonialAuthorsToShowOnThisPage.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return testimonialAuthorsToShowOnThisPage.indexOf(a.quoteAuthorName) - testimonialAuthorsToShowOnThisPage.indexOf(b.quoteAuthorName);
    });

    let testimonialsWithVideoLinks = _.filter(filteredTestimonialsForThisPage, (testimonial)=>{
      return testimonial.youtubeVideoUrl;
    });

    // Get articles with a showOnTestimonialsPageWithEmoji meta tag to display on this page.
    let articles = sails.config.builtStaticContent.markdownPages.filter((page)=>{
      if(_.startsWith(page.htmlId, 'articles')) {
        return page;
      }
    });

    return {
      testimonials: filteredTestimonialsForThisPage,
      testimonialsWithVideoLinks,
    };

  }


};

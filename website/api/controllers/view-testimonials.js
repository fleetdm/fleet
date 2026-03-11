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


    let sortOrderOfTestimonialAuthorsShownOnThisPage = [
      'Scott MacVicar',
      'Nick Fohs',
      'Wes Whetstone',
      'Erik Gomez',
      'Matt Carr',
      'nico waisman',
      'Kenny Botelho',
      'Dan Grzelak',
      'Eric Tan',
    ];

    let testimonialAuthorsToExcludeOnThisPage = [
      'Alvaro Gutierrez',
      'Joe Pistone',
      'Brendan Shaklovitz',
      'Abubakar Yousafzai',
      'Dhruv Majumdar',
      'matt carr',
      'Charles Zaffery',
      'Tom Larkin',// Note: excluded becasue we already show a quote from this person
      'Nico Waisman',// Note: excluded becasue we already show a quote from this person
    ];

    let filteredTestimonialsForThisPage = _.filter(testimonials, (testimonial)=>{
      return !testimonialAuthorsToExcludeOnThisPage.includes(testimonial.quoteAuthorName);
    });

    filteredTestimonialsForThisPage.sort((a, b)=>{
      if(sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(a.quoteAuthorName) === -1){
        return 1;
      } else if(sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(b.quoteAuthorName) === -1) {
        return -1;
      }
      return sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(a.quoteAuthorName) - sortOrderOfTestimonialAuthorsShownOnThisPage.indexOf(b.quoteAuthorName);
    });

    let testimonialsWithVideoLinks = _.filter(filteredTestimonialsForThisPage, (testimonial)=>{
      return testimonial.youtubeVideoUrl;
    });


    return {
      testimonials: filteredTestimonialsForThisPage,
      testimonialsWithVideoLinks,
    };

  }


};

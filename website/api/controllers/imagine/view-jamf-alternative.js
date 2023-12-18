module.exports = {


  friendlyName: 'View jamf alternative',


  description: 'Display "Jamf alternative" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/imagine/jamf-alternative'
    }

  },


  fn: async function () {
    // Get testimonials for the <scrolalble-tweets> component.
    let testimonialsForScrollableTweets = sails.config.builtStaticContent.testimonials;
    // Respond with view.
    return {
      testimonialsForScrollableTweets,
    };
  }


};

module.exports = {


  friendlyName: 'View jamf migration',


  description: 'Display the in-depth "Jamf migration" landing page, which holds the previous /lp/replace-jamf content.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/jamf-migration'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

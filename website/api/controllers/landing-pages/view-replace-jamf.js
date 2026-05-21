module.exports = {


  friendlyName: 'View replace jamf',


  description: 'Display "Replace Jamf" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/landing-pages/replace-jamf'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

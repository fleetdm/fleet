module.exports = {


  friendlyName: 'View profiles',


  description: 'Display "Profiles" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/profiles'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

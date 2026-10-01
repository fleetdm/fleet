module.exports = {


  friendlyName: 'View configuration generator',


  description: 'Display "Configuration generator" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/configuration-generator'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

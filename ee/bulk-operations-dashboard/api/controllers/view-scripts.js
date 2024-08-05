module.exports = {


  friendlyName: 'View scripts',


  description: 'Display "Scripts" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/scripts'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

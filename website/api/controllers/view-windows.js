module.exports = {


  friendlyName: 'View windows',


  description: 'Display "Windows" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/windows'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'View basic documentation',


  description: 'Display "Basic documentation" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/docs/basic-documentation'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

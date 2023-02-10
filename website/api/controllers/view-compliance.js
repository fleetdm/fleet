module.exports = {


  friendlyName: 'View compliance',


  description: 'Display "Compliance" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/compliance'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

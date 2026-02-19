module.exports = {


  friendlyName: 'View reset trial',


  description: 'Display "Reset trial" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/reset-trial'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'View faq',


  description: 'Display "FAQ" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/faq'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

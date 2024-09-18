module.exports = {


  friendlyName: 'View software',


  description: 'Display "Software" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/software/software'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

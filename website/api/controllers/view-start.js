module.exports = {


  friendlyName: 'View start',


  description: 'Display "Start" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/start'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

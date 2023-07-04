module.exports = {


  friendlyName: 'View upgrade',


  description: 'Display "Upgrade" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/upgrade'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

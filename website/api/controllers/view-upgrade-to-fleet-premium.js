module.exports = {


  friendlyName: 'View upgrade to fleet premium',


  description: 'Display "Upgrade to fleet premium" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/upgrade-to-fleet-premium'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

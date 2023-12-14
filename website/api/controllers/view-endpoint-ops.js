module.exports = {


  friendlyName: 'View endpoint ops',


  description: 'Display "Endpoint ops" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/endpoint-ops'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

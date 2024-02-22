module.exports = {


  friendlyName: 'View fleetctl preview',


  description: 'Display "fleetctl preview" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/fleetctl-preview'
    },

    redirect: {
      description: 'The requesting user is not logged in.',
      responseType: 'redirect'
    },

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

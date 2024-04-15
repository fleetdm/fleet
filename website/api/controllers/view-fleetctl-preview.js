module.exports = {


  friendlyName: 'View fleetctl preview',


  description: 'Display "fleetctl preview" page.',

  inputs: {
    start: {
      type: 'boolean',
      description: 'A boolean flag that will hide the "next steps" buttons on the page if set to true',
      defaultsTo: false,
    }
  },

  exits: {

    success: {
      viewTemplatePath: 'pages/fleetctl-preview'
    },

    redirect: {
      description: 'The requesting user is not logged in.',
      responseType: 'redirect'
    },

  },


  fn: async function ({start}) {

    // Respond with view.
    return {hideNextStepsButtons: start};

  }


};

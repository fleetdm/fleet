module.exports = {


  friendlyName: 'View state of device management',


  description: 'Display "State of device management" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/reports/state-of-device-management'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

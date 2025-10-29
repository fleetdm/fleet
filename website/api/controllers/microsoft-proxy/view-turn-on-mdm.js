module.exports = {


  friendlyName: 'View turn on mdm',


  description: 'Display "Turn on mdm" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/microsoft-proxy/turn-on-mdm'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

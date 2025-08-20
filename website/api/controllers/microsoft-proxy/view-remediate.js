module.exports = {


  friendlyName: 'View remediate',


  description: 'Display "Remediate" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/microsoft-proxy/remediate'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

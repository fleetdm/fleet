module.exports = {


  friendlyName: 'View how Fleet fits your stack',


  description: 'Display "How Fleet fits into your IT stack" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/how-fleet-fits-your-stack'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

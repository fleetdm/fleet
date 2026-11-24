module.exports = {


  friendlyName: 'View apple',


  description: 'Display "Apple" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/apple'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

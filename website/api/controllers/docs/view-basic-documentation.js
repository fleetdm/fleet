module.exports = {


  friendlyName: 'View basic documentation',


  description: 'Display "Basic documentation" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/docs/basic-documentation'
    }

  },


  fn: async function () {

    // Serve appropriate doc page content.
    // > Inspired by https://github.com/sailshq/sailsjs.com/blob/b53c6e6a90c9afdf89e5cae00b9c9dd3f391b0e7/api/controllers/documentation/view-documentation.js
    // TODO

    // Respond with view.
    return {};

  }


};

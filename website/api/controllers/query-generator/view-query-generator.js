module.exports = {


  friendlyName: 'View query generator',


  description: 'Display "Query generator" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/query-generator'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

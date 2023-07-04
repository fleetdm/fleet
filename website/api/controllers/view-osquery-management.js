module.exports = {


  friendlyName: 'View osquery management',


  description: 'Display "Osquery management" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/osquery-management'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'View sandbox waitlist',


  description: 'Display "Sandbox waitlist" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/sandbox-waitlist'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'View jamf alternative',


  description: 'Display "Jamf alternative" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/imagine/jamf-alternative'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

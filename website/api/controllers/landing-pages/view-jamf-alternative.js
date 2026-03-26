module.exports = {


  friendlyName: 'View jamf alternative',


  description: 'Display the "Jamf Alternative" landing page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/jamf-alternative'
    }

  },


  fn: async function () {

    // No additional data needed for this static landing page.
    return {};

  }


};

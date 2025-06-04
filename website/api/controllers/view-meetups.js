module.exports = {


  friendlyName: 'View meetups',


  description: 'Display "Meetups" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/meetups'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

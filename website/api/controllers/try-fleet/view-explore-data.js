module.exports = {


  friendlyName: 'View explore data',


  description: 'Display "Explore data" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/explore-data'
    }

  },


  fn: async function () {

    // Respond with view.
    return {};

  }


};

module.exports = {


  friendlyName: 'Prepare sprint',


  description: 'Prepare the product and engineering kanban boards for the next release.',


  extendedDescription: 'Usage: `sails run prepare script 4.14.0`',


  args: ['upcomingReleaseVersion'],


  inputs: {

    upcomingReleaseVersion: {
      type: 'string',
      example: '4.14.0',
      description: 'The version of the upcoming release whose development is about to begin.',
      custom: (upcomingReleaseVersion)=> upcomingReleaseVersion.match(/^[0-9]+\.[0-9]+(\.0)?$/)
    },

  },


  fn: async function () {

    sails.log('Running custom shell script... (`sails run prepare-sprint`)');

    // For all issues in "✅ Ready for release":
    // • close them
    // • remove from project
    // • set milestone for all issues (why? clean archive)
    // TODO

    // For all issues in the "Product" board's "🦤 Estimated" column but only those in the next release:
    // • move to "✅ In development"
    // • add to "Release" (engineering) board in "🥚 Ready" column
    // TODO

    // Log the breakdown of estimations, column by column, on the release board from before this script was run versus now, after it has run successfully.
    // TODO

  }


};


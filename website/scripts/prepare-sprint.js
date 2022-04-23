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

    // Ensure milestone exists in both fleetdm/fleet and fleetdm/confidential
    // TODO

    // List milestones · https://docs.github.com/en/rest/issues/milestones#list-milestones
    // Create milestone · https://docs.github.com/en/rest/issues/milestones#create-a-milestone



    // For all issues in "Engineering" board's "✅ Ready for release":
    // • close them
    // • remove from project
    // • set milestone for all issues (why? clean archive)
    // TODO

    // Close issue · https://docs.github.com/en/rest/issues/issues#update-an-issue
    // Remove card from project · https://docs.github.com/en/issues/trying-out-the-new-projects-experience/using-the-api-to-manage-projects#deleting-an-item-from-a-project
    // Set milestone · https://docs.github.com/en/rest/issues/issues#update-an-issue



    // For all issues in the "Product" board's "✅ In development" column tagged with the next release (as opposed to the only other possible option: the release after that):
    // • move to "✅ In development"
    // • add to "Release" (engineering) board in "🥚 Ready" column
    // TODO

    // Update select field ("status") · https://docs.github.com/en/issues/trying-out-the-new-projects-experience/using-the-api-to-manage-projects#updating-a-single-select-field
    // Add card to project · https://docs.github.com/en/issues/trying-out-the-new-projects-experience/using-the-api-to-manage-projects#adding-an-item-to-a-project


    // Log the breakdown of estimations, column by column, on the release board from before this script was run versus now, after it has run successfully.
    // TODO

  }


};

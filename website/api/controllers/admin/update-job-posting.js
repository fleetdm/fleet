module.exports = {


  friendlyName: 'Update job posting',


  description: 'Open or close a role on fleetdm.com/hire.',


  inputs: {

    jobPostingId: { type: 'number', required: true },

    isOpen: { type: 'boolean', required: true },

  },


  exits: {

    notFound: { responseType: 'notFound' },

  },


  fn: async function ({jobPostingId, isOpen}) {

    let updatedJobPosting = await JobPosting.updateOne({id: jobPostingId}).set({isOpen});
    if(!updatedJobPosting) {
      throw 'notFound';
    }

  }


};

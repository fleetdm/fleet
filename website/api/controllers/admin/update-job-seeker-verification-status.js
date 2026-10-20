module.exports = {


  friendlyName: 'Update job seeker verification status',


  description: 'Record whether a Fleet team member confirmed a job seeker\'s eligibility claim.',


  inputs: {

    jobSeekerId: { type: 'number', required: true },

    verificationStatus: {
      type: 'string',
      required: true,
      isIn: ['Pending', 'Verified', 'Not eligible'],
    },

  },


  exits: {

    notFound: { responseType: 'notFound' },

  },


  fn: async function ({jobSeekerId, verificationStatus}) {

    let updatedJobSeeker = await JobSeeker.updateOne({id: jobSeekerId}).set({verificationStatus});
    if(!updatedJobSeeker) {
      throw 'notFound';
    }

  }


};

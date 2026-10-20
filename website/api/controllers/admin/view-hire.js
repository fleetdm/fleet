module.exports = {


  friendlyName: 'View hire',


  description: 'Display "Hire" admin page, where Fleet team members post roles and review job seekers.',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/hire'
    }

  },


  fn: async function () {

    let jobPostings = await JobPosting.find()
    .populate('interestedJobSeekers', {select: ['id']})
    .sort('createdAt DESC');

    let jobSeekers = await JobSeeker.find({isLooking: true})
    .populate('user')
    .populate('interestedInJobPostings', {select: ['id', 'title', 'companyName']})
    .sort('createdAt DESC');

    // Only send what the page needs about each user.
    jobSeekers = jobSeekers.map((jobSeeker)=>({
      ...jobSeeker,
      user: _.pick(jobSeeker.user, ['id', 'firstName', 'lastName', 'emailAddress', 'organization']),
    }));

    return {
      jobPostings,
      jobSeekers,
      jobBoardRoles: sails.config.custom.jobBoardRoles,
    };

  }


};

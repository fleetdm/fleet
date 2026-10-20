module.exports = {


  friendlyName: 'View hire',


  description: 'Display "Hire" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/hire'
    }

  },


  fn: async function () {

    let openJobPostings = await JobPosting.find({isOpen: true})
    .select(['title', 'role', 'companyName', 'location', 'description', 'createdAt'])
    .sort('createdAt DESC');

    let myJobSeekerProfile;
    if(this.req.me) {
      myJobSeekerProfile = await JobSeeker.findOne({user: this.req.me.id})
      .populate('interestedInJobPostings', {select: ['id']});
    }

    return {
      openJobPostings,
      myJobSeekerProfile: myJobSeekerProfile ? _.omit(myJobSeekerProfile, 'crmMatchSummary') : undefined,
      jobBoardRoles: sails.config.custom.jobBoardRoles,
    };

  }


};

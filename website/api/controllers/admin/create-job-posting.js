module.exports = {


  friendlyName: 'Create job posting',


  description: 'Post a new role to fleetdm.com/hire.',


  inputs: {

    title: { type: 'string', required: true, maxLength: 200 },

    role: { type: 'string', required: true },

    companyName: { type: 'string', required: true, maxLength: 200 },

    salesforceAccountId: {
      type: 'string',
      description: 'Set when the company was picked from Salesforce autocomplete.',
    },

    location: { type: 'string', maxLength: 200 },

    description: { type: 'string', maxLength: 5000 },

  },


  exits: {

    success: {
      outputType: {},
    },

    invalidRole: {
      description: 'The provided role is not a role on the Fleet job board.',
      responseType: 'badRequest',
    },

  },


  fn: async function ({title, role, companyName, salesforceAccountId, location, description}) {

    if(!_.contains(sails.config.custom.jobBoardRoles, role)) {
      throw 'invalidRole';
    }

    let newJobPosting = await JobPosting.create({
      title,
      role,
      companyName,
      salesforceAccountId,
      location,
      description,
      postedBy: this.req.me.id,
    }).fetch();

    return newJobPosting;

  }


};

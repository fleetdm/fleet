module.exports = {


  friendlyName: 'View sandbox waitlist',


  description: 'Display "Sandbox waitlist" page.',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/sandbox-waitlist'
    }

  },


  fn: async function () {

    let usersCurrentlyOnWaitlist = await User.find({inSandboxWaitlist: true})
    .sort('createdAt ASC');

    return {
      usersWaitingForSandboxInstance: usersCurrentlyOnWaitlist
    };

  }


};

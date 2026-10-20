module.exports = {


  friendlyName: 'Get superadmins',


  description: 'Display "Superadmins" page, an admin tool for viewing and managing which users have superadmin access to the Fleet website.',


  exits: {

    success: {
      viewTemplatePath: 'pages/admin/superadmins'
    }

  },


  fn: async function () {

    // Only select the attributes that the page needs, so that password hashes, tokens, etc. are never sent to the browser.
    let superAdmins = await User.find({
      where: {isSuperAdmin: true},
      select: ['id', 'firstName', 'lastName', 'emailAddress', 'lastSeenAt', 'createdAt'],
    })
    .sort('emailAddress ASC');

    return {
      superAdmins,
      currentUserId: this.req.me.id,
    };

  }


};

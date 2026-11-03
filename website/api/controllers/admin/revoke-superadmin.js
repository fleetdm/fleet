module.exports = {


  friendlyName: 'Revoke superadmin',


  description: 'Remove superadmin access to the Fleet website from a user (e.g., when someone leaves the company). The user\'s account is not deleted.',


  inputs: {
    userId: {
      type: 'number',
      description: 'The database ID of the superadmin whose superadmin access will be removed.',
      required: true,
    },
  },


  exits: {
    success: {description: 'The user\'s superadmin access was removed.'},
    userNotFound: {description: 'No user account was found with the provided ID.', responseType: 'notFound'},
    notSuperAdmin: {description: 'This user is not a superadmin.', responseType: 'badRequest'},
    cannotRevokeOwnAccess: {description: 'Superadmins cannot remove their own superadmin access.', responseType: 'badRequest'},
    lastSuperAdmin: {description: 'This user is the last remaining superadmin.', responseType: 'badRequest'},
  },


  fn: async function ({userId}) {

    // Note: The `is-super-admin` policy guarantees that `this.req.me` is a superadmin.
    if(userId === this.req.me.id) {
      throw 'cannotRevokeOwnAccess';
    }

    let userToRevoke = await User.findOne({
      where: {id: userId},
      select: ['id', 'emailAddress', 'isSuperAdmin'],
    });

    if(!userToRevoke) {
      throw 'userNotFound';
    }

    if(!userToRevoke.isSuperAdmin) {
      throw 'notSuperAdmin';
    }

    // Never allow the website to be left without any superadmins.
    let numSuperAdmins = await User.count({isSuperAdmin: true});
    if(numSuperAdmins <= 1) {
      throw 'lastSuperAdmin';
    }

    await User.updateOne({id: userToRevoke.id}).set({isSuperAdmin: false});

    sails.log.warn(`Superadmin access was revoked from user #${userToRevoke.id} (${userToRevoke.emailAddress}) by superadmin #${this.req.me.id} (${this.req.me.emailAddress}).`);

  }


};

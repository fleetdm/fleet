module.exports = {


  friendlyName: 'Grant superadmin',


  description: 'Give superadmin access to the Fleet website to an existing user.',


  inputs: {
    emailAddress: {
      type: 'string',
      description: 'The email address of the existing user who will become a superadmin.',
      required: true,
    },
  },


  exits: {
    success: {description: 'The user was granted superadmin access.'},
    userNotFound: {description: 'No user account was found using the provided email address.', responseType: 'notFound'},
    alreadySuperAdmin: {description: 'This user is already a superadmin.', responseType: 'badRequest'},
  },


  fn: async function ({emailAddress}) {

    let userToGrant = await User.findOne({
      where: {emailAddress: emailAddress.toLowerCase()},
      select: ['id', 'isSuperAdmin'],
    });

    if(!userToGrant) {
      throw 'userNotFound';
    }

    if(userToGrant.isSuperAdmin) {
      throw 'alreadySuperAdmin';
    }

    await User.updateOne({id: userToGrant.id}).set({isSuperAdmin: true});

    sails.log.warn(`Superadmin access was granted to user #${userToGrant.id} (${emailAddress.toLowerCase()}) by superadmin #${this.req.me.id} (${this.req.me.emailAddress}).`);

    // Return only the non-sensitive attributes that the page needs to display the new superadmin.
    return await User.findOne({
      where: {id: userToGrant.id},
      select: ['id', 'firstName', 'lastName', 'emailAddress', 'lastSeenAt', 'createdAt'],
    });

  }


};

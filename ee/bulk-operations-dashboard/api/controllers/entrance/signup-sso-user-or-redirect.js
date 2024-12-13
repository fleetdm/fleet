module.exports = {


  friendlyName: 'Signup SSO user or redirect',


  description: 'Looks up or creates user records for Entra SSO users, and attaches the database id to the requesting user\'s session.',


  exits: {
    redirect: {
      responseType: 'redirect',
    },
  },


  fn: async function () {
    if(!this.req.session) {// If the requesting user does not have a session, redirect them to the login page.
      throw {redirect: '/login'};
    }
    // If the requesting user has a session, but it does not contain a ssoUserInformation object, we'll redirect them to the login page.
    if (!this.req.session.ssoUserInformation) {
      throw {redirect: '/login'};
    }
    let ssoUserInfo = this.req.session.ssoUserInformation;

    let possibleUserRecordForThisEntraUser = await User.findOne({emailAddress: ssoUserInfo.unique_name});

    if(possibleUserRecordForThisEntraUser) {
      // If we found an existing user record that uses this Entra user's email address, we'll set the requesting session.userId to be the id of the database record.
      this.req.session.userId = possibleUserRecordForThisEntraUser.id;
    } else {
      // If we did not find a user in the database for this Entra user, we'll create a new one.
      let newUserRecord = await User.create({
        fullName: ssoUserInfo.given_name +' '+ssoUserInfo.family_name,
        emailAddress: ssoUserInfo.unique_name,
        password: await sails.helpers.passwords.hashPassword(ssoUserInfo.sub),// Note: this password cannot be changed.
        // apiToken: await sails.helpers.strings.uuid(),
      }).fetch();
      this.req.session.userId = newUserRecord.id;
    }
    // Redirect the logged-in user to the homepage.
    return this.res.redirect('/');

  }


};

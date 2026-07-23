module.exports = {


  friendlyName: 'Signup sso user or redirect',


  description: 'Looks up or creates user records for SSO users, and attaches the database id to the requesting user\'s session.',


  exits: {
    redirect: {
      responseType: 'redirect',
    },
  },


  fn: async function () {
    // If the sso hook is not configured, redirect the user to the login page.
    if(!sails.config.custom.ssoClientSecret) {
      throw {redirect: '/login'};
    }
    if(!this.req.session) {// If the requesting user does not have a session, redirect them to the login page.
      throw {redirect: '/login'};
    }
    // If the requesting user has a session, but it does not contain a passport object, we'll redirect them to the login page.
    if (!this.req.session.passport || !this.req.session.passport.user) {
      throw {redirect: '/login'};
    }

    let ssoUserInfo = this.req.session.passport.user.userinfo;
    let possibleUserRecordForThisSsoUser = await User.findOne({emailAddress: ssoUserInfo.email});

    if(possibleUserRecordForThisSsoUser) {
      // If we found an existing user record that uses this SSO user's email address, we'll set the requesting session.userId to be the id of the database record.
      this.req.session.userId = possibleUserRecordForThisSsoUser.id;
    } else {
      // If we did not find a user in the database for this SSO user, we'll create a new one.
      let newUserRecord = await User.create({
        fullName: ssoUserInfo.name,
        emailAddress: ssoUserInfo.email,
        password: await sails.helpers.passwords.hashPassword(ssoUserInfo.sub),// Note: this password cannot be changed.
      }).fetch();
      this.req.session.userId = newUserRecord.id;
    }
    // Redirect the logged-in user to the homepage.
    return this.res.redirect('/');

  }


};

module.exports = {


  friendlyName: 'Signup okta user or redirect',


  description: 'Looks up or creates user records for Okta SSO users, and attaches the database id to the requesting user\'s session.',


  exits: {
    redirect: {
      responseType: 'redirect',
    },
  },


  fn: async function () {

    if(!this.req.session) {// If the requesting user does not have a session, redirect them to the login page.
      throw {redirect: '/login'};
    }
    // If the requesting user has a session, but it does not contain a passport object, we'll redirect them to the login page.
    if (!this.req.session.passport.user) {
      throw {redirect: '/login'};
    }

    let oktaUserInfo = this.req.session.passport.user.userinfo;
    let possibleUserRecordForThisOktaUser = await User.findOne({emailAddress: oktaUserInfo.preferred_username});

    if(possibleUserRecordForThisOktaUser) {
      // If we found an existing user record that uses this Okta user's email address, we'll set the requesting session.userId to be the id of the database record.
      this.req.session.userId = possibleUserRecordForThisOktaUser.id;
    } else {
      // If we did not find a user in the database for this OktaSSO user, we'll create a new one.
      let newUserRecord = await User.create({
        fullName: oktaUserInfo.given_name +' '+oktaUserInfo.family_name,
        emailAddress: oktaUserInfo.preferred_username,
        password: await sails.helpers.passwords.hashPassword(oktaUserInfo.sub),// Note: this password cannot be changed.
      }).fetch();
      this.req.session.userId = newUserRecord.id;
    }
    // Redirect the logged-in user to the homepage.
    return this.res.redirect('/');

  }


};

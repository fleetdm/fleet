module.exports = {


  friendlyName: 'Confirm email',


  description:
`Confirm a new user's email address, or an existing user's request for an email address change,
then redirect to either a special landing page (for newly-signed up users), or the account page
(for existing users who just changed their email address).`,


  inputs: {

    token: {
      description: 'The confirmation token from the email.',
      example: '4-32fad81jdaf$329'
    }

  },


  exits: {

    success: {
      description: 'Email address confirmed and requesting user logged in.'
    },

    redirect: {
      description: 'Email address confirmed and requesting user logged in.  Since this looks like a browser, redirecting...',
      responseType: 'redirect'
    },

    invalidOrExpiredToken: {
      responseType: 'expired',
      description: 'The provided token is expired, invalid, or already used up.',
    },

    emailAddressNoLongerAvailable: {
      statusCode: 409,
      viewTemplatePath: '500',
      description: 'The email address is no longer available.',
      extendedDescription: 'This is an edge case that is not always anticipated by websites and APIs.  Since it is pretty rare, the 500 server error page is used as a simple catch-all.  If this becomes important in the future, this could easily be expanded into a custom error page or resolution flow.  But for context: this behavior of showing the 500 server error page mimics how popular apps like Slack behave under the same circumstances.',
    }

  },


  fn: async function ({token}) {

    // If no token was provided, this is automatically invalid.
    if (!token) {
      throw 'invalidOrExpiredToken';
    }

    // Get the user with the matching email token.
    var user = await User.findOne({ emailProofToken: token });

    // If no such user exists, or their token is expired, bail.
    if (!user || user.emailProofTokenExpiresAt <= Date.now()) {
      throw 'invalidOrExpiredToken';
    }

    if (user.emailStatus === 'unconfirmed') {
      //  ┌─┐┌─┐┌┐┌┌─┐┬┬─┐┌┬┐┬┌┐┌┌─┐  ╔═╗╦╦═╗╔═╗╔╦╗ ╔╦╗╦╔╦╗╔═╗  ╦ ╦╔═╗╔═╗╦═╗  ┌─┐┌┬┐┌─┐┬┬
      //  │  │ ││││├┤ │├┬┘││││││││ ┬  ╠╣ ║╠╦╝╚═╗ ║───║ ║║║║║╣   ║ ║╚═╗║╣ ╠╦╝  ├┤ │││├─┤││
      //  └─┘└─┘┘└┘└  ┴┴└─┴ ┴┴┘└┘└─┘  ╚  ╩╩╚═╚═╝ ╩   ╩ ╩╩ ╩╚═╝  ╚═╝╚═╝╚═╝╩╚═  └─┘┴ ┴┴ ┴┴┴─┘
      // If this is a new user confirming their email for the first time,
      // then just update the state of their user record in the database,
      // store their user id in the session (just in case they aren't logged
      // in already), and then redirect them to the "email confirmed" page.
      await User.updateOne({ id: user.id }).set({
        emailStatus: 'confirmed',
        emailProofToken: '',
        emailProofTokenExpiresAt: 0
      });
      this.req.session.userId = user.id;

      if (this.req.wantsJSON) {
        return;
      } else {
        throw { redirect: '/email/confirmed' };
      }

    } else if (user.emailStatus === 'change-requested') {
      //  ┌─┐┌─┐┌┐┌┌─┐┬┬─┐┌┬┐┬┌┐┌┌─┐  ╔═╗╦ ╦╔═╗╔╗╔╔═╗╔═╗╔╦╗  ┌─┐┌┬┐┌─┐┬┬
      //  │  │ ││││├┤ │├┬┘││││││││ ┬  ║  ╠═╣╠═╣║║║║ ╦║╣  ║║  ├┤ │││├─┤││
      //  └─┘└─┘┘└┘└  ┴┴└─┴ ┴┴┘└┘└─┘  ╚═╝╩ ╩╩ ╩╝╚╝╚═╝╚═╝═╩╝  └─┘┴ ┴┴ ┴┴┴─┘
      if (!user.emailChangeCandidate){
        throw new Error(`Consistency violation: Could not update Stripe customer because this user record's emailChangeCandidate ("${user.emailChangeCandidate}") is missing.  (This should never happen.)`);
      }

      // Last line of defense: since email change candidates are not protected
      // by a uniqueness constraint in the database, it's important that we make
      // sure no one else managed to grab this email in the mean time since we
      // last checked its availability. (This is a relatively rare edge case--
      // see exit description.)
      if (await User.count({ emailAddress: user.emailChangeCandidate }) > 0) {
        throw 'emailAddressNoLongerAvailable';
      }

      // If billing features are enabled, also update the billing email for this
      // user's linked customer entry in the Stripe API to make sure they receive
      // email receipts.
      // > Note: If there was not already a Stripe customer entry for this user,
      // > then one will be set up implicitly, so we'll need to persist it to our
      // > database.  (This could happen if Stripe credentials were not configured
      // > at the time this user was originally created.)
      if(sails.config.custom.enableBillingFeatures) {
        let didNotAlreadyHaveCustomerId = (! user.stripeCustomerId);
        let stripeCustomerId = await sails.helpers.stripe.saveBillingInfo.with({
          stripeCustomerId: user.stripeCustomerId,
          emailAddress: user.emailChangeCandidate
        }).timeout(5000).retry();
        if (didNotAlreadyHaveCustomerId){
          await User.updateOne({ id: user.id }).set({
            stripeCustomerId
          });
        }
      }

      // Finally update the user in the database, store their id in the session
      // (just in case they aren't logged in already), then redirect them to
      // their "my account" page so they can see their updated email address.
      await User.updateOne({ id: user.id })
      .set({
        emailStatus: 'confirmed',
        emailProofToken: '',
        emailProofTokenExpiresAt: 0,
        emailAddress: user.emailChangeCandidate,
        emailChangeCandidate: '',
      });
      this.req.session.userId = user.id;
      if (this.req.wantsJSON) {
        return;
      } else {
        throw { redirect: '/account' };
      }

    } else {
      throw new Error(`Consistency violation: User ${user.id} has an email proof token, but somehow also has an emailStatus of "${user.emailStatus}"!  (This should never happen.)`);
    }

  }


};

module.exports = {


  friendlyName: 'Send password recovery email',


  description: 'Send a password recovery notification to the user with the specified email address.',


  inputs: {

    emailAddress: {
      description: 'The email address of the alleged user who wants to recover their password.',
      example: 'rydahl@example.com',
      type: 'string',
      required: true
    },

    passwordResetRequestedFrom: {
      type: 'string',
      description: 'The section of the website that the user has requested the password reset from.',
      defaultsTo: 'customers',
    }

  },


  exits: {

    success: {
      description: 'The email address might have matched a user in the database.  (If so, a recovery email was sent.)'
    },

  },


  fn: async function ({emailAddress, passwordResetRequestedFrom}) {

    // Find the record for this user.
    // (Even if no such user exists, pretend it worked to discourage sniffing.)
    var userRecord = await User.findOne({ emailAddress });
    if (!userRecord) {
      return;
    }//•
    let passwordResetLocation = '/customers/new-password';
    if(passwordResetRequestedFrom === 'sandbox'){
      passwordResetLocation = '/try-fleet/new-password';
    }
    // Come up with a pseudorandom, probabilistically-unique token for use
    // in our password recovery email.
    var token = await sails.helpers.strings.random('url-friendly');

    // Store the token on the user record
    // (This allows us to look up the user when the link from the email is clicked.)
    await User.updateOne({ id: userRecord.id })
    .set({
      passwordResetToken: token,
      passwordResetTokenExpiresAt: Date.now() + sails.config.custom.passwordResetTokenTTL,
    });

    // Send recovery email
    await sails.helpers.sendTemplateEmail.with({
      to: emailAddress,
      subject: 'Password reset instructions',
      template: 'email-reset-password',
      templateData: {
        passwordResetLocation: passwordResetLocation,
        token: token
      }
    });

  }


};

module.exports = {


  friendlyName: 'View new password',


  description: 'Display "New password" page.',


  inputs: {

    token: {
      description: 'The password reset token from the email.',
      example: '4-32fad81jdaf$329'
    }

  },


  exits: {

    success: {
      viewTemplatePath: 'pages/entrance/new-password'
    },

    invalidOrExpiredToken: {
      responseType: 'expired',
      description: 'The provided token is expired, invalid, or has already been used.',
    }

  },


  fn: async function ({token}) {

    // If password reset token is missing, display an error page explaining that the link is bad.
    if (!token) {
      sails.log.warn('Attempting to view new password (recovery) page, but no reset password token included in request!  Displaying error page...');
      throw 'invalidOrExpiredToken';
    }//•

    // Look up the user with this reset token.
    var userRecord = await User.findOne({ passwordResetToken: token });
    // If no such user exists, or their token is expired, display an error page explaining that the link is bad.
    if (!userRecord || userRecord.passwordResetTokenExpiresAt <= Date.now()) {
      throw 'invalidOrExpiredToken';
    }

    // Grab token and include it in view locals
    return {
      token,
    };

  }


};

module.exports = {


  friendlyName: 'Update password',


  description: 'Update the password for the logged-in user.',


  inputs: {

    oldPassword: {
      description: 'The new, unencrypted password.',
      example: 'abc123v2',
      required: true
    },
    newPassword: {
      description: 'The new, unencrypted password.',
      example: 'abc123v2',
      required: true
    }

  },

  exits: {
    success: {
      description: 'The requesting user agent has been successfully changed their password.',
    },

    badPassword: {
      description: `The provided password does not match the user's current password.`,
      responseType: 'unauthorized'
    }
  },


  fn: async function (inputs) {

    await sails.helpers.passwords.checkPassword(inputs.oldPassword, this.req.me.password)
    .intercept('incorrect', 'badPassword');

    // Hash the new password.
    var hashed = await sails.helpers.passwords.hashPassword(inputs.newPassword);

    // Update the record for the logged-in user.
    await User.updateOne({ id: this.req.me.id })
    .set({
      password: hashed
    });

  }


};

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
    },

    couldNotChangeSandboxPassword: {
      description: 'An error occurred while changing this user\'s password on their Fleet sandbox instance',
    }
  },


  fn: async function (inputs) {

    await sails.helpers.passwords.checkPassword(inputs.oldPassword, this.req.me.password)
    .intercept('incorrect', 'badPassword');

    // Hash the new password.
    var hashed = await sails.helpers.passwords.hashPassword(inputs.newPassword);

    // If this user has a valid provisioned Fleet Sandbox instance, we'll update their password there as well.
    if(this.req.me.fleetSandboxURL) {

      // If the user's Fleet Sandbox instance is still valid, we'll use their old password to get the authorization token from the sandbox and use it to change their password.
      if(this.req.me.fleetSandboxExpiresAt < Date.now()) {
        // Get the record for this user to update their Fleet Sandbox password using their old hashed password.
        let userRecord = await User.findOne({ id: this.req.me.id });

        // Send a post request to the `/login` endpoint of the Fleet Sandbox instance to get this user's api token.
        let authToken = await sails.helpers.http.post(userRecord.fleetSandboxURL+'/api/v1/fleet/login', {
          'email': userRecord.emailAddress,
          'password': userRecord.password
        }).intercept('non200Response', 'couldNotChangeSandboxPassword');

        // If we received a token in the response from the Fleet Sandbox instance, we'll use that to send a POST request to the `/change_password` endpoint to update this users password with the hashed version of their fleetdm.com password
        if(!authToken.token) {
          throw 'couldNotChangeSandboxPassword';
        } else {
          // Update the user's password on their fleet instance
          await sails.helpers.http.post(
              userRecord.fleetSandboxURL+'/api/v1/fleet/change_password',
            {
              'old_password': userRecord.password,
              'new_password': hashed,
            },
            {'Authorization': 'Bearer '+authToken.token}
          ).intercept('non200Response', 'couldNotChangeSandboxPassword');
        }
      }
    }

    // Update the record for the logged-in user.
    await User.updateOne({ id: this.req.me.id })
    .set({
      password: hashed
    });

  }


};

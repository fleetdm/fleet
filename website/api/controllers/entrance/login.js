module.exports = {


  friendlyName: 'Login',


  description: 'Log in using the provided email and password combination.',


  extendedDescription:
`This action attempts to look up the user record in the database with the
specified email address.  Then, if such a user exists, it uses
bcrypt to compare the hashed password from the database with the provided
password attempt.`,


  inputs: {

    emailAddress: {
      description: 'The email to try in this attempt, e.g. "irl@example.com".',
      type: 'string',
      required: true
    },

    password: {
      description: 'The unencrypted password to try in this attempt, e.g. "passwordlol".',
      type: 'string',
      required: true
    },

    rememberMe: {
      description: 'Whether to extend the lifetime of the user\'s session.',
      extendedDescription:
`Note that this is NOT SUPPORTED when using virtual requests (e.g. sending
requests over WebSockets instead of HTTP).`,
      type: 'boolean'
    }

  },


  exits: {

    success: {
      description: 'The requesting user agent has been successfully logged in.',
      extendedDescription:
`Under the covers, this stores the id of the logged-in user in the session
as the \`userId\` key.  The next time this user agent sends a request, assuming
it includes a cookie (like a web browser), Sails will automatically make this
user id available as req.session.userId in the corresponding action.  (Also note
that, thanks to the included "custom" hook, when a relevant request is received
from a logged-in user, that user's entire record from the database will be fetched
and exposed as \`req.me\`.)`
    },

    badCombo: {
      description: `The provided email and password combination does not
      match any user in the database.`,
      responseType: 'unauthorized'
      // ^This uses the custom `unauthorized` response located in `api/responses/unauthorized.js`.
      // To customize the generic "unauthorized" response across this entire app, change that file
      // (see api/responses/unauthorized).
      //
      // To customize the response for _only this_ action, replace `responseType` with
      // something else.  For example, you might set `statusCode: 498` and change the
      // implementation below accordingly (see http://sailsjs.com/docs/concepts/controllers).
    },
    noUser: {
      description: `The provided email does not match any user in the database.`,
      responseType: 'unauthorized'
    },

    couldNotProvisionSandbox: {
      description: 'An error occurred while trying to provision the Fleet Sandbox Instance'
    },

  },


  fn: async function ({emailAddress, password, rememberMe}) {

    // Look up by the email address.
    // (note that we lowercase it to ensure the lookup is always case-insensitive,
    // regardless of which database we're using)
    var userRecord = await User.findOne({
      emailAddress: emailAddress.toLowerCase(),
    });

    // If there was no matching user, respond thru the "noUser" exit.
    if(!userRecord) {
      throw 'noUser';
    }

    // If the password doesn't match, then also exit thru "badCombo".
    await sails.helpers.passwords.checkPassword(password, userRecord.password)
    .intercept('incorrect', 'badCombo');

    // If "Remember Me" was enabled, then keep the session alive for
    // a longer amount of time.  (This causes an updated "Set Cookie"
    // response header to be sent as the result of this request -- thus
    // we must be dealing with a traditional HTTP request in order for
    // this to work.)
    if (rememberMe) {
      if (this.req.isSocket) {
        sails.log.warn(
          'Received `rememberMe: true` from a virtual request, but it was ignored\n'+
          'because a browser\'s session cookie cannot be reset over sockets.\n'+
          'Please use a traditional HTTP request instead.'
        );
      } else {
        this.req.session.cookie.maxAge = sails.config.custom.rememberMeCookieMaxAge;
      }
    }//ﬁ

    // If this user does not have a Fleet Sandbox instance, we'll provision them one.
    if(!userRecord.fleetSandboxURL) {
      let fleetSandboxExpiresAt = Date.now() + (24*60*60*1000);

      let cloudProvisionerResponse = await sails.helpers.http.post(
        'https://sandbox.fleetdm.com/new',
        {
          'name': userRecord.firstName + ' ' + userRecord.lastName,
          'email': userRecord.emailAddress,
          'password': userRecord.password,
          'sandbox_expiration': new Date(fleetSandboxExpiresAt).toISOString(), // sending expiration_timestamp as an ISO string.
        },
        {
          'Authorization':sails.config.custom.cloudProvisionerSecret
        }
      )
      .timeout(5000)
      .intercept('non200Response', 'couldNotProvisionSandbox');

      if(!cloudProvisionerResponse.URL) {
        throw 'couldNotProvisionSandbox';
      } else {
        // Update this user's record with the fleetSandboxURL and fleetSandboxExpiresAt
        await User.updateOne({id: userRecord.id}).set({
          fleetSandboxURL: cloudProvisionerResponse.URL,
          fleetSandboxExpiresAt: fleetSandboxExpiresAt,
        });
        // Poll the Fleet Sandbox Instance's /healthz endpoint until it returns a 200 response, if no response was recieved in 10 seconds, we'll throw an error.
        await sails.helpers.flow.until( async()=>{
          let serverResponse = await sails.helpers.http.sendHttpRequest('GET', cloudProvisionerResponse.URL+'/healthz').timeout(5000).tolerate('non200Response').tolerate('requestFailed');
          if(serverResponse && serverResponse.statusCode) {
            return serverResponse.statusCode === 200;
          }
        }, 10000);
      }
    }
    // Modify the active session instance.
    // (This will be persisted when the response is sent.)
    this.req.session.userId = userRecord.id;

  }

};

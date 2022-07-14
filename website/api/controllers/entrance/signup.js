module.exports = {


  friendlyName: 'Signup',


  description: 'Sign up for a new user account.',


  extendedDescription:
`This creates a new user record in the database, signs in the requesting user agent
by modifying its [session](https://sailsjs.com/documentation/concepts/sessions), and
(if emailing with Mailgun is enabled) sends an account verification email.

If a verification email is sent, the new user's account is put in an "unconfirmed" state
until they confirm they are using a legitimate email address (by clicking the link in
the account verification message.)`,


  inputs: {

    emailAddress: {
      required: true,
      type: 'string',
      isEmail: true,
      description: 'The email address for the new account, e.g. m@example.com.',
      extendedDescription: 'Must be a valid email address.',
    },

    password: {
      required: true,
      type: 'string',
      maxLength: 200,
      example: 'passwordlol',
      description: 'The unencrypted password to use for the new account.'
    },

    organization: {
      required: false, //« Change organization to be required: false.
      defaultsTo: '', //« Add a default value
      type: 'string',
      maxLength: 120,
      example: 'The Sails company',
      description: 'The organization the user works for'
    },

    firstName:  {
      required: true,
      type: 'string',
      example: 'Frida',
      description: 'The user\'s first name.',
    },

    lastName:  {
      required: true,
      type: 'string',
      example: 'Rivera',
      description: 'The user\'s last name.',
    },

    // Add optional input
    signupReason: {
      required: false,
      defaultsTo: 'Buy a license',
      type: 'string',
      isIn: ['Buy a license', 'Try Fleet Sandbox'],
    }

  },


  exits: {

    success: {
      description: 'New user account was created successfully.'
    },

    invalid: {
      responseType: 'badRequest',
      description: 'The provided firstName, lastName, organization, password and/or email address are invalid.',
      extendedDescription: 'If this request was sent from a graphical user interface, the request '+
      'parameters should have been validated/coerced _before_ they were sent.'
    },

    emailAlreadyInUse: {
      statusCode: 409,
      description: 'The provided email address is already in use.',
    },

    couldNotProvisionSandbox: {
      description: 'An error occurred while trying to provision the Fleet Sandbox Instance'
    }


  },

  // Add sandboxExpirationTimestamp to inputs
  fn: async function ({emailAddress, password, firstName, lastName, organization, signupReason}) {

    var newEmailAddress = emailAddress.toLowerCase();


    // Build up data for the new user record and save it to the database.
    // (Also use `fetch` to retrieve the new ID so that we can use it below.)
    var newUserRecord = await User.create(_.extend({
      firstName,
      lastName,
      organization,
      signupReason,
      emailAddress: newEmailAddress,
      password: await sails.helpers.passwords.hashPassword(password),
      tosAcceptedByIp: this.req.ip
    }, sails.config.custom.verifyEmailAddresses? {
      emailProofToken: await sails.helpers.strings.random('url-friendly'),
      emailProofTokenExpiresAt: Date.now() + sails.config.custom.emailProofTokenTTL,
      emailStatus: 'unconfirmed'
    }:{}))
    .intercept('E_UNIQUE', 'emailAlreadyInUse')
    .intercept({name: 'UsageError'}, 'invalid')
    .fetch();

    // If billing feaures are enabled, save a new customer entry in the Stripe API.
    // Then persist the Stripe customer id in the database.
    if (sails.config.custom.enableBillingFeatures) {
      let stripeCustomerId = await sails.helpers.stripe.saveBillingInfo.with({
        emailAddress: newEmailAddress
      }).timeout(5000).retry();
      await User.updateOne({id: newUserRecord.id})
      .set({
        stripeCustomerId
      });
    }

    // If "Try Fleet Sandbox" was provided as the signupReason, this is a user signing up to try Fleet Sandbox.
    if(signupReason === 'Try Fleet Sandbox') {

      // Creating an expiration JS timestamp for the Fleet sandbox instance. NOTE: We send this value to the cloud provisioner API as an ISO 8601 string.
      let fleetSandboxExpiresAt = Date.now() + (24*60*60*1000);

      // Send a POST request to the cloud provisioner API
      let cloudProvisionerResponse = await sails.helpers.http.post(sails.config.custom.fleetSandboxProvisionerURL, {
        'name': firstName + ' ' + lastName,
        'email': emailAddress,
        'password': newUserRecord.password, //« Sending the hashed password to the Fleet Sandbox instance
        'sandbox_expiration': new Date(fleetSandboxExpiresAt).toISOString(), // sending expiration_timestamp as an ISO string.
      })
      .timeout(5000)
      .intercept('non200Response', 'couldNotProvisionSandbox');

      if(cloudProvisionerResponse.URL) {
        // Update the user's record with the fleetSandboxURL, fleetSandboxExpiresAt, and fleetSandboxKey.
        await User.updateOne({id: newUserRecord.id}).set({
          fleetSandboxURL: cloudProvisionerResponse.URL,
          fleetSandboxExpiresAt: fleetSandboxExpiresAt,
        });
        // Start polling the /healthz endpoint of the created Fleet Sandbox instance, once it returns a 200 response, we'll continue.
        await sails.helpers.flow.until( async()=>{
          let serverResponse = await sails.helpers.http.sendHttpRequest('GET', cloudProvisionerResponse.URL+'/healthz')
          .timeout(5000)
          .tolerate('non200Response')
          .tolerate('requestFailed');
          if(serverResponse && serverResponse.statusCode) {
            return serverResponse.statusCode === 200;
          }
        });
      } else {
        throw 'couldNotProvisionSandbox';
      }
    }

    // Store the user's new id in their session.
    this.req.session.userId = newUserRecord.id;

    if (sails.config.custom.verifyEmailAddresses) {
      // Send "confirm account" email
      await sails.helpers.sendTemplateEmail.with({
        to: newEmailAddress,
        from: sails.config.custom.fromEmailAddress,
        fromName: sails.config.custom.fromName,
        subject: 'Please confirm your account',
        template: 'email-verify-account',
        templateData: {
          firstName,
          token: newUserRecord.emailProofToken
        }
      });
    } else {
      sails.log.info('Skipping new account email verification... (since `verifyEmailAddresses` is disabled)');
    }
    return;

  }

};

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
      description: 'The unhashed (plain text) password to use for the new account.'
    },

    organization: {
      required: true,
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

    signupReason: {
      type: 'string',
      isIn: ['Buy a license', 'Try Fleet Sandbox'],
      defaultsTo: 'Buy a license',
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

    requestToSandboxTimedOut: {
      statusCode: 408,
      description: 'The request to the cloud provisioner exceeded the set timeout.',
    },

    emailAlreadyInUse: {
      statusCode: 409,
      description: 'The provided email address is already in use.',
    },


  },

  fn: async function ({emailAddress, password, firstName, lastName, organization, signupReason}) {

    if(!sails.config.custom.cloudProvisionerSecret){
      throw new Error('The authorization token for the cloud provisioner API (sails.config.custom.cloudProvisionerSecret) is missing! If you just want to test aspects of fleetdm.com locally, and are OK with the cloud provisioner failing if you try to use it, you can set a fake secret when starting a local server by lifting the server with "sails_custom__cloudProvisionerSecret=test sails lift"');
    }

    if(sails.config.custom.fleetSandboxWaitlistEnabled === undefined){
      throw new Error(`The Fleet Sandbox waitlist configuration variable (sails.config.custom.fleetSandboxWaitlistEnabled) is missing!`);
    }

    var newEmailAddress = emailAddress.toLowerCase();

    // Checking if a user with this email address exists in our database before we send a request to the cloud provisioner.
    if(await User.findOne({emailAddress: newEmailAddress})) {
      throw 'emailAlreadyInUse';
    }


    if (!sails.config.custom.enableBillingFeatures) {
      throw new Error('The Stripe configuration variables (sails.config.custom.stripePublishableKey and sails.config.custom.stripeSecret) are missing!');
    }

    // Create a new customer entry in the Stripe API for this user before we send a request to the cloud provisioner.
    let stripeCustomerId = await sails.helpers.stripe.saveBillingInfo.with({
      emailAddress: newEmailAddress
    })
    .timeout(5000)
    .retry()
    .intercept((error)=>{
      return new Error(`An error occurred when trying to create a Stripe Customer for a new user with the using the email address ${newEmailAddress}. The incomplete user record has not been saved in the database, and the user will be asked to try signing up again. Full error: ${error.raw}`);
    });

    let newUserRecord;

    // If the sandbox waitlist is enabled, we'll create a record without fleetSandboxURL,fleetSandboxExpiresAt,fleetSandboxDemoKey values and with inSandboxWaitlist set to true.
    if(sails.config.custom.fleetSandboxWaitlistEnabled === true) {
      newUserRecord = await User.create(_.extend({
        firstName,
        lastName,
        organization,
        emailAddress: newEmailAddress,
        signupReason,
        password: await sails.helpers.passwords.hashPassword(password),
        stripeCustomerId,
        inSandboxWaitlist: true,
        tosAcceptedByIp: this.req.ip
      }, sails.config.custom.verifyEmailAddresses? {
        emailProofToken: await sails.helpers.strings.random('url-friendly'),
        emailProofTokenExpiresAt: Date.now() + sails.config.custom.emailProofTokenTTL,
        emailStatus: 'unconfirmed'
      }:{}))
      .intercept('E_UNIQUE', 'emailAlreadyInUse')
      .intercept({name: 'UsageError'}, 'invalid')
      .fetch();

    } else {
      // If the Fleet Sandbox waitlist is not enabled (sails.config.custom.fleetSandboxWaitlistEnabled) We'll provision a Sandbox instance BEFORE creating the new User record.
      // This way, if this fails, we won't save the new record to the database, and the user will see an error on the signup form asking them to try again.

      const FIVE_DAYS_IN_MS = (5*24*60*60*1000);
      // Creating an expiration JS timestamp for the Fleet sandbox instance. NOTE: We send this value to the cloud provisioner API as an ISO 8601 string.
      let fleetSandboxExpiresAt = Date.now() + FIVE_DAYS_IN_MS;

      // Creating a fleetSandboxDemoKey, this will be used for the user's password when we log them into their Sandbox instance.
      let fleetSandboxDemoKey = await sails.helpers.strings.uuid();

      // Send a POST request to the cloud provisioner API
      let cloudProvisionerResponseData = await sails.helpers.http.post(
        'https://sandbox.fleetdm.com/new',
        { // Request body
          'name': firstName + ' ' + lastName,
          'email': newEmailAddress,
          'password': fleetSandboxDemoKey, //« this provisioner API was originally designed to accept passwords, but rather than specifying the real plaintext password, since users always access Fleet Sandbox from their fleetdm.com account anyway, this generated demo key is used instead to avoid any confusion
          'sandbox_expiration': new Date(fleetSandboxExpiresAt).toISOString(), // sending expiration_timestamp as an ISO string.
        },
        { // Request headers
          'Authorization':sails.config.custom.cloudProvisionerSecret
        }
      )
      .timeout(10000)// FUTURE: set this timeout to be 5000ms
      .intercept(['requestFailed', 'non200Response'], (err)=>{
        // If we received a non-200 response from the cloud provisioner API, we'll throw a 500 error.
        return new Error('When attempting to provision a new user who just signed up ('+emailAddress+'), the cloud provisioner gave a non 200 response. The incomplete user record has not been saved in the database, and the user will be asked to try signing up again. Raw response received from provisioner: '+err.stack);
      })
      .intercept({name: 'TimeoutError'},(err)=>{
        // If the request timed out, log a warning and return a 'requestToSandboxTimedOut' response.
        sails.log.warn('When attempting to provision a new user who just signed up ('+emailAddress+'), the request to the cloud provisioner took over timed out. The incomplete user record has not been saved in the database, and the user will be asked to try signing up again. Raw error: '+err.stack);
        return 'requestToSandboxTimedOut';
      });

      if(!cloudProvisionerResponseData.URL) {
        // If we didn't receive a URL in the response from the cloud provisioner API, we'll throwing an error before we save the new user record and the user will need to try to sign up again.
        throw new Error(
          `When provisioning a Fleet Sandbox instance for a new user who just signed up (${emailAddress}), the response data from the cloud provisioner API was malformed. It did not contain a valid Fleet Sandbox instance URL in its expected "URL" property.
          The incomplete user record has not been saved in the database, and the user will be asked to try signing up again.
          Here is the malformed response data (parsed response body) from the cloud provisioner API: ${cloudProvisionerResponseData}`
        );
      }

      // If "Try Fleet Sandbox" was provided as the signupReason, we'll make sure their Sandbox instance is live before we continue.
      if(signupReason === 'Try Fleet Sandbox') {
        // Start polling the /healthz endpoint of the created Fleet Sandbox instance, once it returns a 200 response, we'll continue.
        await sails.helpers.flow.until( async()=>{
          let healthCheckResponse = await sails.helpers.http.sendHttpRequest('GET', cloudProvisionerResponseData.URL+'/healthz')
          .timeout(5000)
          .tolerate('non200Response')
          .tolerate('requestFailed')
          .tolerate({name: 'TimeoutError'});
          if(healthCheckResponse) {
            return true;
          }
        }, 10000).intercept('tookTooLong', ()=>{
          return new Error('This newly provisioned Fleet Sandbox instance (for '+emailAddress+') is taking too long to respond with a 2xx status code, even after repeatedly polling the health check endpoint.  Note that failed requests and non-2xx responses from the health check endpoint were ignored during polling.  Search for a bit of non-dynamic text from this error message in the fleetdm.com source code for more info on exactly how this polling works.');
        });
      }

      // Build up data for the new user record and save it to the database.
      // (Also use `fetch` to retrieve the new ID so that we can use it below.)
      newUserRecord = await User.create(_.extend({
        firstName,
        lastName,
        organization,
        emailAddress: newEmailAddress,
        signupReason,
        password: await sails.helpers.passwords.hashPassword(password),
        fleetSandboxURL: cloudProvisionerResponseData.URL,
        fleetSandboxExpiresAt,
        fleetSandboxDemoKey,
        stripeCustomerId,
        inSandboxWaitlist: false,
        tosAcceptedByIp: this.req.ip
      }, sails.config.custom.verifyEmailAddresses? {
        emailProofToken: await sails.helpers.strings.random('url-friendly'),
        emailProofTokenExpiresAt: Date.now() + sails.config.custom.emailProofTokenTTL,
        emailStatus: 'unconfirmed'
      }:{}))
      .intercept('E_UNIQUE', 'emailAlreadyInUse')
      .intercept({name: 'UsageError'}, 'invalid')
      .fetch();

      // Send a POST request to Zapier
      await sails.helpers.http.post(
        'https://hooks.zapier.com/hooks/catch/3627242/bqsf4rj/',
        {
          'emailAddress': newEmailAddress,
          'organization': organization,
          'firstName': firstName,
          'lastName': lastName,
          'signupReason': signupReason,
          'webhookSecret': sails.config.custom.zapierSandboxWebhookSecret
        }
      )
      .timeout(5000)
      .tolerate(['non200Response', 'requestFailed', {name: 'TimeoutError'}], (err)=>{
        // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
        sails.log.warn(`When a new user signed up, a lead/contact could not be verified in the CRM for this email address: ${newEmailAddress}. Raw error: ${err}`);
        return;
      });

    }//ﬁ

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

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

    emailAlreadyInUse: {
      statusCode: 409,
      description: 'The provided email address is already in use.',
    },


  },

  fn: async function ({emailAddress, password, firstName, lastName, organization, signupReason}) {

    var newEmailAddress = emailAddress.toLowerCase();

    // Provisioning a Fleet sandbox instance for the new user. Note: Because this is the only place where we provision Sandbox instances, We'll provision a Sandbox instance BEFORE
    // creating the new User record. This way, if this fails, we won't save the new record to the database, and the user will see an error on the signup form asking them to try again.

    // Creating an expiration JS timestamp for the Fleet sandbox instance. NOTE: We send this value to the cloud provisioner API as an ISO 8601 string.
    let fleetSandboxExpiresAt = Date.now() + (24*60*60*1000);

    // Creating a fleetSandboxDemoKey, this will be used for the user's password when we log them into their Sandbox instance.
    let fleetSandboxDemoKey = await sails.helpers.strings.uuid();

    // Send a POST request to the cloud provisioner API
    let cloudProvisionerResponse = await sails.helpers.http.post(
      'https://sandbox.fleetdm.com/new',
      { // Request body
        'name': firstName + ' ' + lastName,
        'email': newEmailAddress,
        'password': fleetSandboxDemoKey,
        'sandbox_expiration': new Date(fleetSandboxExpiresAt).toISOString(), // sending expiration_timestamp as an ISO string.
      },
      { // Request headers
        'Authorization':sails.config.custom.cloudProvisionerSecret
      }
    )
    .timeout(5000)
    .intercept('non200Response', ()=>{
      // If we recieved a non-200 response from the Cloud Provisioner API, we'll throw a 500 error.
      throw new Error('When attempting to provision a new user\'s Fleet Sandbox instance, the Cloud provisioner gave a non 200 response. The incomplete user record has not been saved in the database, and the user will be asked to try signing up again.');
    });

    if(!cloudProvisionerResponse.URL) {
      // If we didn't receive a URL in the response from the Cloud Provisioner API, we'll throwing an error before we save the new user record and the user will need to try to sign up again.
      throw new Error(
        `When provisioning a Fleet Sandbox instance for this new user, the 200 response from the Cloud Provisioner API did not contain a URL of a Sandbox instance.
        The newly created User record has been deleted, and the user will be asked to try signing up again.
        Here is the response from the Cloud Provisioner API: ${cloudProvisionerResponse}`
      );
    }

    // If "Try Fleet Sandbox" was provided as the signupReason, this is a user signing up to try Fleet Sandbox, and we'll make sure their Sandbox instance is live before we continue.
    if(signupReason === 'Try Fleet Sandbox') {
      // Start polling the /healthz endpoint of the created Fleet Sandbox instance, once it returns a 200 response, we'll continue.
      await sails.helpers.flow.until( async()=>{
        let serverResponse = await sails.helpers.http.sendHttpRequest('GET', cloudProvisionerResponse.URL+'/healthz')
        .timeout(5000)
        .tolerate('non200Response')
        .tolerate('requestFailed');
        if(serverResponse && serverResponse.statusCode) {
          return serverResponse.statusCode === 200;
        }
      }, 10000).intercept('tookTooLong', ()=>{
        throw new Error('This newly provisioned Fleet Sandbox instance is taking too long to respond.');
      });
    }

    // Build up data for the new user record and save it to the database.
    // (Also use `fetch` to retrieve the new ID so that we can use it below.)
    var newUserRecord = await User.create(_.extend({
      firstName,
      lastName,
      organization,
      emailAddress: newEmailAddress,
      password: await sails.helpers.passwords.hashPassword(password),
      fleetSandboxURL: cloudProvisionerResponse.URL,
      fleetSandboxExpiresAt,
      fleetSandboxDemoKey,
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

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
      isIn: ['Buy a license', 'Try Fleet'],
      defaultsTo: 'Buy a license',
    },

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

    invalidEmailDomain: {
      description: 'This email address is on a denylist of domains and cannot be used to signup for a fleetdm.com account.',
      responseType: 'badRequest'
    },


  },

  fn: async function ({emailAddress, password, firstName, lastName, organization, signupReason}) {
    // Note: in Oct. 2023, the Fleet Sandbox related code was removed from this action. For more details, see https://github.com/fleetdm/fleet/pull/14638/files

    var newEmailAddress = emailAddress.toLowerCase();
    // Checking if a user with this email address exists in our database before we send a request to the cloud provisioner.
    if(await User.findOne({emailAddress: newEmailAddress})) {
      throw 'emailAlreadyInUse';
    }
    // Check the user's email address and return an 'invalidEmailDomain' response if the domain is in the bannedEmailDomainsForSignup array.
    let emailDomain = newEmailAddress.split('@')[1];
    let bannedEmailDomainsForSignup = [
      'gmail.com',
      'yahoo.com',
      'yahoo.co.uk',
      'hotmail.com',
      'hotmail.co.uk',
      'outlook.com',
      'icloud.com',
      'proton.me',
      'live.com',
      'yandex.ru',
      'ymail.com',
    ];
    if(_.includes(bannedEmailDomainsForSignup, emailDomain)){
      throw 'invalidEmailDomain';
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
    let newUserRecord = await User.create(_.extend({
      firstName,
      lastName,
      organization,
      emailAddress: newEmailAddress,
      signupReason,
      password: await sails.helpers.passwords.hashPassword(password),
      stripeCustomerId,
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
    await sails.helpers.http.post.with({
      url: 'https://hooks.zapier.com/hooks/catch/3627242/30bq2ib/',
      data: {
        newEmailAddress,
        firstName,
        lastName,
        organization,
        signupReason,
        webhookSecret: sails.config.custom.zapierSandboxWebhookSecret,
      }
    })
    .timeout(5000)
    .tolerate(['non200Response', 'requestFailed'], (err)=>{
      // Note that Zapier responds with a 2xx status code even if something goes wrong, so just because this message is not logged doesn't mean everything is hunky dory.  More info: https://github.com/fleetdm/fleet/pull/6380#issuecomment-1204395762
      sails.log.warn(`When a user submitted a contact form message, a lead/contact could not be updated in the CRM for this email address: ${newEmailAddress}. Raw error: ${err}`);
      return;
    });

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

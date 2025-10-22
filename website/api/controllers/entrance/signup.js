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
      description: 'The organization the user works for',
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
    // Check the user's email address and return an 'invalidEmailDomain' response if the domain is in the sails.config.custom.bannedEmailDomainsForWebsiteSubmissions array.
    let emailDomain = newEmailAddress.split('@')[1];
    if(_.includes(sails.config.custom.bannedEmailDomainsForWebsiteSubmissions, emailDomain)){
      throw 'invalidEmailDomain';
    }

    // If organization was not provided (e.g., The user signed up with the signup modal), use their email domain as their organization.
    if(!organization) {
      organization = emailDomain;
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


    // Enrich the information provided.
    let enrichmentInformation = await sails.helpers.iq.getEnriched.with({
      firstName,
      lastName,
      organization,
      emailAddress: newEmailAddress,
    }).tolerate((err)=>{
      sails.log.warn(`When a new user signed up for an account, enrichment information could not be obtained with the information provided. Full error: ${require('util').inspect(err)}`);
      return { employer: undefined, person: undefined};
    });


    let fleetPremiumTrialType = 'local-trial';
    if(enrichmentInformation.employer) {
      if(enrichmentInformation.employer.numberOfEmployees > 700){
        fleetPremiumTrialType = 'render-trial';
      }
    }
    // Test code to make the

    let thirtyDaysFromNowAt = Date.now() + (1000 * 60 * 60 * 24 * 30);
    let trialLicenseKeyForThisUser = await sails.helpers.createLicenseKey.with({
      numberOfHosts: 10,
      organization: emailDomain,
      expiresAt: thirtyDaysFromNowAt,
    });

    await User.updateOne({id: newUserRecord.id}).set({
      fleetPremiumTrialLicenseKeyExpiresAt: thirtyDaysFromNowAt,
      fleetPremiumTrialLicenseKey: trialLicenseKeyForThisUser,
      fleetPremiumTrialType,
    });



    if(fleetPremiumTrialType === 'render-trial') {

      // Since this user is eligable for a Render trial, we'll complete the setup process on the trial instance.

      let renderInstancesThatCanBeAssignedToThisUser = await RenderProofOfValue.find({
        where: {status: 'ready-for-assignment', user: null},
        sort: 'createdAt DESC',
      });

      if(!renderInstancesThatCanBeAssignedToThisUser){
        // TODO: what if there are no instances available?
        // Option 1: We run a background helper at this point to provision them a Render instance, redirect them to the /try page after signup, where they have a message saying that their instance will be available in ~5 minutes.
        // ^ We may need to do that last part even if there is an instance available. It takes ~1-2 minutes to apply the
      } else {
        let instanceToAssign = renderInstancesThatCanBeAssignedToThisUser[0];
        // Configure this user's license key on their Fleet instance.
        await sails.helpers.http.sendHttpRequest.with({
          method: 'PUT',
          url: `https://api.render.com/v1/services/${instanceToAssign.renderFleetServiceId}/env-vars/FLEET_LICENSE_KEY`,
          body: {
            key: 'FLEET_LICENSE_KEY',
            value: trialLicenseKeyForThisUser
          },
          headers: {
            authorization: `Bearer ${sails.config.custom.renderApiToken}`
          },
        });
        await sails.helpers.http.sendHttpRequest.with({
          method: 'PUT',
          url: `https://api.render.com/v1/services/${instanceToAssign.renderFleetServiceId}/env-vars/FLEET_LICENSE_KEY`,
          body: {
            key: 'FLEET_LICENSE_KEY',
            value: trialLicenseKeyForThisUser
          },
          headers: {
            authorization: `Bearer ${sails.config.custom.renderApiToken}`
          },
        });

        let startDeployResponse = await sails.helpers.http.sendHttpRequest.with({
          method: 'POST',
          url: `https://api.render.com/v1/services/${instanceToAssign.renderFleetServiceId}/deploys`,
          headers: {
            authorization: `Bearer ${sails.config.custom.renderApiToken}`
          },
        });

        console.log(startDeployResponse);
        let parsedResponseBody = JSON.parse(startDeployResponse.body);

        await RenderProofOfValue.updateOne({id: instanceToAssign.id}).set({
          renderTrialEndsAt: thirtyDaysFromNowAt,
          renderBeforeFirstUseDeployId: parsedResponseBody.id,
          user: newUserRecord.id,
        });
      }

      await sails.helpers.sendTemplateEmail.with({
        to: newEmailAddress,
        from: sails.config.custom.fromEmailAddress,
        fromName: sails.config.custom.fromName,
        subject: 'Your Fleet trial is ready',
        template: 'email-14-day-fleet-premium-trial-started',
        layout: 'layout-nurture-email',
        templateData: {
          firstName,
          emailAddress: newEmailAddress,
        }
      });

    } else {

      await sails.helpers.sendTemplateEmail.with({
        to: newEmailAddress,
        from: sails.config.custom.fromEmailAddress,
        fromName: sails.config.custom.fromName,
        subject: 'Your 30-day Fleet Premium trial key',
        template: 'email-30-day-fleet-premium-trial-started',
        layout: 'layout-nurture-email',
        templateData: {
          firstName,
          emailAddress: newEmailAddress,
        }
      });


    }





    let psychologicalStageChangeReason;
    if(this.req.session.adAttributionString && this.req.session.visitedSiteFromAdAt) {
      let sevenDaysAgoAt = Date.now() - (1000 * 60 * 60 * 24 * 7);
      // If this user visited the website from an ad, set the psychologicalStageChangeReason to be the adCampaignId stored in their session.
      if(this.req.session.visitedSiteFromAdAt > sevenDaysAgoAt) {
        psychologicalStageChangeReason = this.req.session.adAttributionString;
      }
    }
    sails.helpers.salesforce.updateOrCreateContactAndAccount.with({
      emailAddress: newEmailAddress,
      firstName: firstName,
      lastName: lastName,
      organization: emailDomain,// TODO.
      contactSource: 'Website - Sign up',
      psychologicalStageChangeReason,
    }).exec((err)=>{
      if(err){
        sails.log.warn(`Background task failed: When a user (email: ${newEmailAddress} signed up for a fleetdm.com account, a Contact and Account record could not be created/updated in the CRM.`, err);
      }
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

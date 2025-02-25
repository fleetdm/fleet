/**
 * User.js
 *
 * A user who can log in to this application.
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝

    emailAddress: {
      type: 'string',
      required: true,
      unique: true,
      isEmail: true,
      maxLength: 200,
      example: 'mary.sue@example.com'
    },

    emailStatus: {
      type: 'string',
      isIn: ['unconfirmed', 'change-requested', 'confirmed'],
      defaultsTo: 'confirmed',
      description: 'The confirmation status of the user\'s email address.',
      extendedDescription:
        `Users might be created as "unconfirmed" (e.g. normal signup) or as "confirmed" (e.g. hard-coded
        admin users).  When the email verification feature is enabled, new users created via the
        signup form have \`emailStatus: 'unconfirmed'\` until they click the link in the confirmation email.
        Similarly, when an existing user changes their email address, they switch to the "change-requested"
        email status until they click the link in the confirmation email.`
    },

    emailChangeCandidate: {
      type: 'string',
      isEmail: true,
      description: 'A still-unconfirmed email address that this user wants to change to (if relevant).'
    },

    password: {
      type: 'string',
      required: true,
      description: 'Securely hashed representation of the user\'s login password.',
      protect: true,
      example: '2$28a8eabna301089103-13948134nad'
    },

    firstName: {
      type: 'string',
      required: true,
      description: 'The user\'s first name.',
      maxLength: 120,
      example: 'Mary'
    },

    lastName: {
      type: 'string',
      required: true,
      description: 'The user\'s last name.',
      maxLength: 120,
      example: 'van der McHenst'
    },

    organization: {
      type: 'string',
      description: 'The organization the user works for.',
      maxLength: 120,
      example: 'The Sails Company',
    },

    isSuperAdmin: {
      type: 'boolean',
      description: 'Whether this user is a "super admin" with extra permissions, etc.',
      extendedDescription:
        `Super admins might have extra permissions, see a different default home page when they log in,
        or even have a completely different feature set from normal users.  In this app, the \`isSuperAdmin\`
        flag is just here as a simple way to represent two different kinds of users.  Usually, it's a good idea
        to keep the data model as simple as possible, only adding attributes when you actually need them for
        features being built right now.

        For example, a "super admin" user for a small to medium-sized e-commerce website might be able to
        change prices, deactivate seasonal categories, add new offerings, and view live orders as they come in.
        On the other hand, for an e-commerce website like Walmart.com that has undergone years of development
        by a large team, those administrative features might be split across a few different roles.

        So, while this \`isSuperAdmin\` demarcation might not be the right approach forever, it's a good place to start.`
    },

    passwordResetToken: {
      type: 'string',
      description: 'A unique token used to verify the user\'s identity when recovering a password.  Expires after 1 use, or after a set amount of time has elapsed.'
    },

    passwordResetTokenExpiresAt: {
      type: 'number',
      description: 'A JS timestamp (epoch ms) representing the moment when this user\'s `passwordResetToken` will expire (or 0 if the user currently has no such token).',
      example: 1502844074211
    },

    emailProofToken: {
      type: 'string',
      description: 'A pseudorandom, probabilistically-unique token for use in our account verification emails.'
    },

    emailProofTokenExpiresAt: {
      type: 'number',
      description: 'A JS timestamp (epoch ms) representing the moment when this user\'s `emailProofToken` will expire (or 0 if the user currently has no such token).',
      example: 1502844074211
    },

    stripeCustomerId: {
      type: 'string',
      protect: true,
      description: 'The id of the customer entry in Stripe associated with this user (or empty string if this user is not linked to a Stripe customer -- e.g. if billing features are not enabled).',
      extendedDescription:
`Just because this value is set doesn't necessarily mean that this user has a billing card.
It just means they have a customer entry in Stripe, which might or might not have a billing card.`
    },

    hasBillingCard: {
      type: 'boolean',
      description: 'Whether this user has a default billing card hooked up as their payment method.',
      extendedDescription:
`More specifically, this indcates whether this user record's linked customer entry in Stripe has
a default payment source (i.e. credit card).  Note that a user have a \`stripeCustomerId\`
without necessarily having a billing card.`
    },

    billingCardBrand: {
      type: 'string',
      example: 'Visa',
      description: 'The brand of this user\'s default billing card (or empty string if no billing card is set up).',
      extendedDescription: 'To ensure PCI compliance, this data comes from Stripe, where it reflects the user\'s default payment source.'
    },

    billingCardLast4: {
      type: 'string',
      example: '4242',
      description: 'The last four digits of the card number for this user\'s default billing card (or empty string if no billing card is set up).',
      extendedDescription: 'To ensure PCI compliance, this data comes from Stripe, where it reflects the user\'s default payment source.'
    },

    billingCardExpMonth: {
      type: 'string',
      example: '08',
      description: 'The two-digit expiration month from this user\'s default billing card, formatted as MM (or empty string if no billing card is set up).',
      extendedDescription: 'To ensure PCI compliance, this data comes from Stripe, where it reflects the user\'s default payment source.'
    },

    billingCardExpYear: {
      type: 'string',
      example: '2023',
      description: 'The four-digit expiration year from this user\'s default billing card, formatted as YYYY (or empty string if no credit card is set up).',
      extendedDescription: 'To ensure PCI compliance, this data comes from Stripe, where it reflects the user\'s default payment source.'
    },

    tosAcceptedByIp: {
      type: 'string',
      description: 'The IP (ipv4) address of the request that accepted the terms of service.',
      extendedDescription: 'Useful for certain types of businesses and regulatory requirements (KYC, etc.)',
      moreInfoUrl: 'https://en.wikipedia.org/wiki/Know_your_customer'
    },

    lastSeenAt: {
      type: 'number',
      description: 'A JS timestamp (epoch ms) representing the moment at which this user most recently interacted with the backend while logged in (or 0 if they have not interacted with the backend at all yet).',
      example: 1502844074211
    },

    fleetSandboxURL: {
      type: 'string',
      description: 'The URL of the Fleet sandbox instance that was provisioned for this user',
      example: 'https://billybobcat.sandbox.fleetdm.com',
      extendedDescription: 'As of Oct. 2023, new user records will not have this value set.'
    },

    fleetSandboxExpiresAt: {
      type: 'number',
      description: 'An JS timestamp (epoch ms) representing when this user\'s fleet sandbox instance will expire',
      example: '1502844074211',
      extendedDescription: 'As of Oct. 2023, new user records will not have this value set.'
    },

    fleetSandboxDemoKey: {
      type: 'string',
      description: 'The UUID that is used as the password of this user\'s Fleet Sandbox instance that is generated when the user signs up. Only used to log the user into their Fleet Sandbox instance while it is still live.',
      extendedDescription: 'As of Oct. 2023, new user records will not have this value set.'
    },

    signupReason: {
      type: 'string',
      description: 'The reason this user signed up for a fleetdm.com account',
      isIn: ['Try Fleet Sandbox', 'Buy a license', 'Try Fleet'],
    },

    inSandboxWaitlist: {
      type: 'boolean',
      description: 'whether this user is on the Fleet Sandbox waitlist.',
      defaultsTo: false
    },

    primaryBuyingSituation: {
      type: 'string',
      description: 'The primary buying situation the user selected when they signed up.',
      extendedDescription: 'User records created before 2024-03-14 will have this attribute set to ""',
      isIn: [
        'eo-security',
        'eo-it',
        'mdm',
        'vm',
      ]
    },

    lastSubmittedGetStartedQuestionnaireStep: {
      type: 'string',
      description: 'The last step the user reached in the get started form.',
      defaultsTo: 'start',
    },

    getStartedQuestionnaireAnswers: {
      type: 'json',
      description: 'This answers the user provided when they filled out the get started form.',
      defaultsTo: {},
    },

    psychologicalStage: {
      type: 'string',
      description: 'This user\'s psychological stage based on the answers to the get started questionnaire.',
      isIn: [
        '1 - Unaware',
        '2 - Aware',
        '3 - Intrigued',
        '4 - Has use case',
        '5 - Personally confident',
        '6 - Has team buy-in'
      ],
      defaultsTo: '2 - Aware'
    },

    psychologicalStageLastChangedAt: {
      type: 'number',
      description: 'A JS timestamp of when this user\'s psychological stage changed.',
      extendedDescription: 'Used when deciding whether or not to send a nuture email to this user',
    },

    stageThreeNurtureEmailSentAt: {
      type: 'number',
      description: 'A JS timestamp of when the stage 3 nurture email was sent to the user, or 1 if the user is unsubscribed from automated emails.',
    },

    stageFourNurtureEmailSentAt: {
      type: 'number',
      description: 'A JS timestamp of when the stage 4 nurture email was sent to the user, or 1 if the user is unsubscribed from automated emails.',
    },

    stageFiveNurtureEmailSentAt: {
      type: 'number',
      description: 'A JS timestamp of when the stage 5 nurture email was sent to the user, or 1 if the user is unsubscribed from automated emails.',
    },

    fleetPremiumTrialLicenseKey: {
      type: 'string',
      description: 'A Fleet Premium license key that was generated for this user when they progressed through the get started questionnaire.',
    },

    fleetPremiumTrialLicenseKeyExpiresAt: {
      type: 'number',
      description: 'A JS timestamp of when this user\'s Fleet Premium trial license key expires.',
    },

    canUseQueryGenerator: {
      type: 'boolean',
      description: 'Whether or not this user can access the query generator page',
      defaultsTo: false,
    },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝
    // n/a

    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝
    // n/a

  },


};

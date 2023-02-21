/**
 * Custom configuration
 * (sails.config.custom)
 *
 * One-off settings specific to your application.
 *
 * For more information on custom configuration, visit:
 * https://sailsjs.com/config/custom
 */

module.exports.custom = {

  /**************************************************************************
  *                                                                         *
  * The base URL to use during development.                                 *
  *                                                                         *
  * • No trailing slash at the end                                          *
  * • `http://` or `https://` at the beginning.                             *
  *                                                                         *
  * > This is for use in custom logic that builds URLs.                     *
  * > It is particularly handy for building dynamic links in emails,        *
  * > but it can also be used for user-uploaded images, webhooks, etc.      *
  *                                                                         *
  **************************************************************************/
  baseUrl: 'http://localhost:2024',

  /**************************************************************************
  *                                                                         *
  * The TTL (time-to-live) for various sorts of tokens before they expire.  *
  *                                                                         *
  **************************************************************************/
  passwordResetTokenTTL: 24*60*60*1000,// 24 hours
  emailProofTokenTTL:    24*60*60*1000,// 24 hours

  /**************************************************************************
  *                                                                         *
  * The extended length that browsers should retain the session cookie      *
  * if "Remember Me" was checked while logging in.                          *
  *                                                                         *
  **************************************************************************/
  rememberMeCookieMaxAge: 30*24*60*60*1000, // 30 days

  /**************************************************************************
  *                                                                         *
  * Automated email configuration                                           *
  *                                                                         *
  * Sandbox Sendgrid credentials for use during development, as well as any *
  * other default settings related to "how" and "where" automated emails    *
  * are sent.                                                               *
  *                                                                         *
  * (https://app.sendgrid.com/settings/api_keys)                            *
  *                                                                         *
  **************************************************************************/
  // sendgridSecret: 'SG.fake.3e0Bn0qSQVnwb1E4qNPz9JZP5vLZYqjh7sn8S93oSHU',
  //--------------------------------------------------------------------------
  // /\  Configure this to enable support for automated emails.
  // ||  (Important for password recovery, verification, contact form, etc.)
  //--------------------------------------------------------------------------

  // The sender that all outgoing emails will appear to come from.
  fromEmailAddress: 'noreply@example.com',
  fromName: 'The NEW_APP_NAME Team',

  // Email address for receiving support messages & other correspondences.
  // > If you're using the default privacy policy, this will be referenced
  // > as the contact email of your "data protection officer" for the purpose
  // > of compliance with regulations such as GDPR.
  internalEmailAddress: 'support+development@example.com',

  // Whether to require proof of email address ownership any time a new user
  // signs up, or when an existing user attempts to change their email address.
  verifyEmailAddresses: false,

  /**************************************************************************
  *                                                                         *
  * Billing & payments configuration                                        *
  *                                                                         *
  * (https://dashboard.stripe.com/account/apikeys)                          *
  *                                                                         *
  **************************************************************************/
  // stripePublishableKey: 'pk_test_Zzd814nldl91104qor5911gjald',
  // stripeSecret: 'sk_test_Zzd814nldl91104qor5911gjald',
  //--------------------------------------------------------------------------
  // /\  Configure these to enable support for billing features.
  // ||  (Or if you don't need billing, feel free to remove them.)
  //--------------------------------------------------------------------------

  /***************************************************************************
  *                                                                          *
  * Directly responsible individuals (DRIs) whose changes to areas of the    *
  * code respository (outside of the core product code) are auto-approved,   *
  * even during code freezes.                                                *
  *                                                                          *
  * See api/controllers/webhooks/receive-from-github.js for context.         *
  *                                                                          *
  ***************************************************************************/
  githubRepoDRIByPath: {// fleetdm/fleet
    'README.md': ['mikermcneil', 'jarodreyes', 'mike-j-thomas', 'zwass'],// (github brandfront)
    'tools/fleetctl-npm/README.md': ['mikermcneil', 'mike-j-thomas', 'jarodreyes', 'zwass'],//« brandfront for fleetctl package on npm

    'CODEOWNERS': ['zwass', 'mikermcneil'],
    '.github/workflows': ['zwass', 'mikermcneil'],// (misc GitHub Actions. Note that some are also addressed more specifically below in relevant sections)

    // GitHub issue templates
    '.github/ISSUE_TEMPLATE': ['mikermcneil', 'lukeheath', 'hollidayn'],
    '.github/ISSUE_TEMPLATE/bug-report.md': ['xpkoala','zhumo','noahtalerman', 'lukeheath'],
    '.github/ISSUE_TEMPLATE/feature-request.md': ['xpkoala', 'zhumo','noahtalerman', 'lukeheath'],
    '.github/ISSUE_TEMPLATE/smoke-tests.md': ['xpkoala', 'zhumo','lukeheath','noahtalerman', 'lukeheath'],

    'articles': ['jarodreyes', 'mike-j-thomas', 'eashaw', 'zwass', 'mikermcneil'],

    'handbook': ['mike-j-thomas', 'eashaw', 'mikermcneil', 'zwass', 'charlottechance'],// (default for handbook)
    'handbook/company': 'mikermcneil',
    'handbook/business-operations': ['hollidayn', 'charlottechance'],
    'handbook/engineering': 'zwass',
    'handbook/product': ['noahtalerman', 'zhumo'],
    'handbook/security': 'mikermcneil',
    'handbook/customers': ['alexmitchelliii','zayhanlon','dherder'],
    'handbook/marketing': ['jarodreyes', 'mike-j-thomas'],

    'website': 'mikermcneil',// (default for website)
    'website/views': 'eashaw',
    'website/assets': 'eashaw',
    'website/package.json': 'eashaw',
    '.github/workflows/deploy-fleet-website.yml': ['eashaw','mikermcneil'],// (website deploy script)
    '.github/workflows/test-website.yml': ['eashaw','mikermcneil'],// (website CI test script)

    'website/config/routes.js': ['eashaw', 'mike-j-thomas', 'jarodreyes'],// (for managing website URLs)

    'website/api/controllers/imagine': ['eashaw', 'jarodreyes'],// landing pages

    'docs': ['zwass', 'mikermcneil', 'zhumo', 'jarodreyes', 'ksatter'],// (default for docs)
    'docs/images': ['noahtalerman', 'eashaw', 'mike-j-thomas'],
    'docs/Using-Fleet/REST-API.md': ['ksatter','lukeheath'],
    'docs/Contributing/API-for-contributors.md': ['ksatter','lukeheath'],
    'docs/Deploying/FAQ.md': ['ksatter'],
    'docs/Contributing/FAQ.md': ['ksatter'],
    'docs/Using-Fleet/FAQ.md': ['ksatter'],

    'docs/01-Using-Fleet/standard-query-library/standard-query-library.yml': ['mikermcneil','zhumo','eashaw','lucasmrod','sharon-fdm','artemist-work','marcosd4h'],// (standard query library)
    'schema': ['zhumo','eashaw','zwass','mikermcneil','lucasmrod','sharon-fdm','artemist-work','marcosd4h'],// (Osquery table schema)
  },

  confidentialGithubRepoDRIByPath: {// fleetdm/confidential

    // Folders of configuration files
    'mdm_profiles': ['lukeheath', 'zwass'],
    'vpn': ['rfairburn', 'zwass'],

    // Folder that any fleetie (team member contracted with company) can push to, willy-nilly
    'free-for-all': '*',

    // Boilerplate
    'README.md': ['mikermcneil', 'zwass', 'charlottechance', 'hollidayn'],
    'CODEOWNERS': ['mikermcneil', 'zwass', 'charlottechance', 'hollidayn', 'dherder', 'zayhanlon'],
    '.gitignore': ['mikermcneil', 'zwass', 'charlottechance', 'hollidayn', 'dherder', 'zayhanlon'],

    // CI/CD workflows
    '.github': ['mikermcneil', 'zwass', 'charlottechance', 'hollidayn'],

    // GitHub issue templates
    '.github/ISSUE_TEMPLATE': ['mikermcneil', 'zwass', 'zayhanlon', 'hollidayn', 'alexmitchelliii', 'dherder'],
    '.github/ISSUE_TEMPLATE/2-website-changes.md': 'mike-j-thomas',
    '.github/ISSUE_TEMPLATE/3-opportunity Fleet Premium PoV.md': 'alexmitchelliii',
    '.github/ISSUE_TEMPLATE/3-sale.md': 'alexmitchelliii',
    '.github/ISSUE_TEMPLATE/4-release.md': ['noahtalerman', 'zwass', 'zhumo'],
    '.github/ISSUE_TEMPLATE/5-monthly-accounting.md': 'hollidayn',
    '.github/ISSUE_TEMPLATE/6-speaking-event.md': ['mike-j-thomas', 'jarodreyes'],
    '.github/ISSUE_TEMPLATE/9-renewal.md': ['zayhanlon', 'hollidayn', 'alexmitchelliii'],
    '.github/ISSUE_TEMPLATE/hiring.md': 'charlottechance',
    '.github/ISSUE_TEMPLATE/onboarding.md': 'charlottechance',
    '.github/ISSUE_TEMPLATE/y-offboarding.md': 'charlottechance',
    '.github/ISSUE_TEMPLATE/x-moving.md': ['charlottechance'],
    '.github/ISSUE_TEMPLATE/equity-grants.md': ['charlottechance','hollidayn'],
    '.github/ISSUE_TEMPLATE/signature-or-legal-review.md': ['hollidayn'],
    '.github/ISSUE_TEMPLATE/new-fleet-instance.md': ['charlottechance','hollidayn', 'zayhanlon'],

  },


  /***************************************************************************
  *                                                                          *
  * Any other custom config this Sails app should use during development.    *
  * (and possibly in ALL environments, if not overridden in config/env/)     *
  *                                                                          *
  ***************************************************************************/
  // Contact form:
  // slackWebhookUrlForContactForm: '…',

  // GitHub bot:
  // githubAccessToken: '…',
  // githubBotWebhookSecret: '…',
  // slackWebhookUrlForGithubBot: '…',
  // mergeFreezeAccessToken: '…',

  //…

};

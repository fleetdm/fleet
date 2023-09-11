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

  //  ██████╗ ██████╗ ██╗███████╗
  //  ██╔══██╗██╔══██╗██║██╔════╝
  //  ██║  ██║██████╔╝██║███████╗
  //  ██║  ██║██╔══██╗██║╚════██║
  //  ██████╔╝██║  ██║██║███████║
  //  ╚═════╝ ╚═╝  ╚═╝╚═╝╚══════╝
  //
  /***************************************************************************
  *                                                                          *
  * If a PR contains changes within one of these paths, the DRI is requested *
  * for approval. (If a higher-level path also has a DRI specified, only the *
  * most specific DRI is requested for approval.)                            *
  *                                                                          *
  * See also the CODEOWNERS file in fleetdm/fleet for more context / links.  *
  *                                                                          *
  ***************************************************************************/
  githubRepoDRIByPath: {// fleetdm/fleet
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // FUTURE: To avoid repeating structure and comments, consolidate all these configs w/ something like:
    //    ````
    //    'articles': { dri: 'mike-j-thomas', maintainers: ['mike-j-thomas', 'mike-j-thomas', 'spokanemac', 'mikermcneil'], repo: 'fleetdm/fleet' },
    //    ````
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    // 🚀 Code for core product and integrations
    'ee/tools/puppet': 'georgekarrv',//« Puppet integration (especially useful with macOS MDM turned on) -- FYI: Originally developed by request from "customer-eponym"

    // ⚗️ Reference, config surface, built-in queries, API, and other documentation
    // 'docs': '',                                      // « Covered in CODEOWNERS (2023-07-22)
    // 'docs/Using-Fleet/REST-API.md': '',              // « Covered in CODEOWNERS (2023-07-22)
    // 'docs/Contributing/API-for-contributors.md': '', // « Covered in CODEOWNERS (2023-07-22)
    // 'schema': '',                                    // « Covered in CODEOWNERS (2023-07-22)
    'docs/01-Using-Fleet/standard-query-library/standard-query-library.yml': 'rachaelshaw', //« Built-in queries
    'ee/cis': 'sharon-fdm',//« Fleet Premium only: built-in queries  (built-in policies for CIS benchmarks)  -- FYI: On 2023-07-15, we changed this so that Sharon, Lucas, Marcos, and Rachel are all maintainers, but where there is a single DRI who is automatically requested approval from.

    // 🫧 Articles and release notes
    'articles': 'spokanemac',
    'CHANGELOG.md': 'lukeheath',

    // 🫧 Website (fleetdm.com)
    'website': 'mikermcneil',// (catch-all)
    'website/assets': 'eashaw', // « Eric is DRI for website frontend code
    'website/views': 'eashaw',
    'website/api': 'mikermcneil',//« Website backend, scripts, deps
    'website/api/controllers/webhooks/receive-from-github.js': 'mikermcneil',// github bot (webhook)
    'website/api/controllers/imagine': 'eashaw',// landing pages
    'website/config': 'mikermcneil',
    'website/config/routes.js': 'eashaw',//« Website redirects and URLs
    'website/scripts': 'mikermcneil',
    'website/package.json': 'eashaw',

    // 🫧 Pricing and features
    // 'website/views/pages/pricing.ejs': '',                // « Covered in CODEOWNERS (2023-07-22)
    // 'handbook/company/pricing-features-table.yml': '',    // « Covered in CODEOWNERS (2023-07-22)

    // 🫧 Other brandfronts
    'README.md': 'mikermcneil',// « GitHub brandfront
    'tools/fleetctl-npm/README.md': 'mikermcneil',// « NPM brandfront (npmjs.com/package/fleetctl)

    // 🦿 Repo automation and change control settings
    // 'CODEOWNERS': 'mikermcneil',// « Covered in CODEOWNERS (2023-08-10)
    'website/config/custom.js': 'mikermcneil',

    // 🦿 Handbook
    'handbook': 'mikermcneil', // See https://github.com/fleetdm/fleet/pull/13195
    //'handbook/company/ceo.md': 'sampfluger88',
    //'handbook/company': 'mikermcneil',
    //'handbook/business-operations': 'mikermcneil',
    //'handbook/engineering': 'lukeheath',
    //'handbook/product': 'zhumo',
    //'handbook/customers': 'alexmitchelliii',
    //'handbook/marketing': 'mikermcneil',

    // 🦿 GitHub issue templates
    // '.github/ISSUE_TEMPLATE': 'mikermcneil',// « Covered in CODEOWNERS (2023-08-10)

  },

  // FUTURE: Support DRIs for confidential and other repos (except see other note above about a consolidated way to do it, to reduce these 4-6 config keys into one)


  //  ███╗   ███╗ █████╗ ██╗███╗   ██╗████████╗ █████╗ ██╗███╗   ██╗███████╗██████╗ ███████╗
  //  ████╗ ████║██╔══██╗██║████╗  ██║╚══██╔══╝██╔══██╗██║████╗  ██║██╔════╝██╔══██╗██╔════╝
  //  ██╔████╔██║███████║██║██╔██╗ ██║   ██║   ███████║██║██╔██╗ ██║█████╗  ██████╔╝███████╗
  //  ██║╚██╔╝██║██╔══██║██║██║╚██╗██║   ██║   ██╔══██║██║██║╚██╗██║██╔══╝  ██╔══██╗╚════██║
  //  ██║ ╚═╝ ██║██║  ██║██║██║ ╚████║   ██║   ██║  ██║██║██║ ╚████║███████╗██║  ██║███████║
  //  ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝╚══════╝
  //
  /***************************************************************************
  *                                                                          *
  * Maintainers whose changes to areas of respositories are auto-approved,   *
  * even during code freezes.                                                *
  *                                                                          *
  * See also the CODEOWNERS file in fleetdm/fleet for more context / links.  *
  *                                                                          *
  ***************************************************************************/
  githubRepoMaintainersByPath: {// fleetdm/fleet

    // Code for core product and integrations
    'ee/tools/puppet': ['roperzh', 'gillespi314', 'mna', 'georgekarrv'],

    // Reference, config surface, built-in queries, API, and other documentation
    'docs': ['rachaelshaw'],// (default for docs)
    'docs/01-Using-Fleet/standard-query-library/standard-query-library.yml': ['rachaelshaw'],// (standard query library)
    'schema': ['rachaelshaw'],// (Osquery table schema)
    'ee/cis': ['sharon-fdm', 'lucasmrod', 'marcosd4h', 'rachelElysia', 'rachaelshaw'],

    // Articles and release notes
    'CHANGELOG.md': ['zwass', 'mikermcneil', 'spokanemac', 'noahtalerman', 'zhumo', 'lukeheath'],
    'articles': ['mike-j-thomas', 'mike-j-thomas', 'eashaw', 'zwass', 'mikermcneil', 'spokanemac'],
    'website/assets/images/articles': ['spokanemac', 'mike-j-thomas', 'mike-j-thomas', 'eashaw', 'zwass', 'mikermcneil'],

    // Website (fleetdm.com)
    'website': 'mikermcneil',// (default for website)
    'website/views': 'eashaw',
    'website/generators': 'eashaw',
    'website/assets': 'eashaw',
    'website/package.json': 'eashaw',
    'website/config/routes.js': ['eashaw', 'mike-j-thomas'],// (for managing website URLs)
    'website/config/policies.js': ['eashaw', 'mikermcneil'],// (for adding new pages and managing permissions)
    'website/api/controllers/imagine': ['eashaw', 'mike-j-thomas'],// landing pages

    // Other brandfronts
    'README.md': ['mikermcneil', 'mike-j-thomas', 'zwass'],//« github brandfront (github.com/fleetdm/fleet)
    'tools/fleetctl-npm/README.md': ['mikermcneil', 'mike-j-thomas', 'zwass'],//« brandfront for fleetctl package on npm (npmjs.com/package/fleetctl)

    // Config as code for infrastructure, internal security and IT use cases, and more.
    //'infrastructure': [],// Decided against in https://github.com/fleetdm/fleet/pull/12890
    //'charts': [], //Decided against in https://github.com/fleetdm/fleet/pull/12890
    //'terraform': [],//Decided against in https://github.com/fleetdm/fleet/pull/12890

    // Github workflows
    '.github/workflows/deploy-fleet-website.yml': ['eashaw','mikermcneil'],// (website deploy script)
    '.github/workflows/test-website.yml': ['eashaw','mikermcneil'],//« website CI test script
    '.github/workflows': ['zwass', 'mikermcneil'],//« CI/CD workflows & misc GitHub Actions. Note that some are also addressed more specifically below in relevant sections)

    // Repo automation and change control settings
    'CODEOWNERS': ['zwass', 'mikermcneil'],// (« for changing who reviews is automatically requested from for given paths)
    'website/config/custom.js': ['eashaw', 'mikermcneil'],// (« for changing whose changes automatically approve and unfreeze relevant PRs changing given paths)

    // Handbook
    'handbook': 'mikermcneil',
    // See https://github.com/fleetdm/fleet/pull/13195
    //'handbook/README.md': 'mikermcneil',
    //'handbook/company': 'mikermcneil',
    //'handbook/company/ceo.md': ['sampfluger88'],
    //'handbook/business-operations': ['jostableford'],
    //'handbook/engineering': ['lukeheath'],
    //'handbook/product': ['zhumo'],
    //'handbook/customers': ['alexmitchelliii'],
    //'handbook/marketing': ['mike-j-thomas'],

    // GitHub issue templates
    '.github/ISSUE_TEMPLATE': ['mikermcneil', 'lukeheath', 'hollidayn'],
    '.github/ISSUE_TEMPLATE/bug-report.md': ['xpkoala','zhumo','noahtalerman', 'lukeheath'],
    '.github/ISSUE_TEMPLATE/feature-request.md': ['xpkoala', 'zhumo','noahtalerman', 'lukeheath'],
    '.github/ISSUE_TEMPLATE/smoke-tests.md': ['xpkoala', 'zhumo','lukeheath','noahtalerman', 'lukeheath'],
  },

  confidentialGithubRepoMaintainersByPath: {// fleetdm/confidential

    // Config as code for infrastructure, internal security and IT use cases, and more.
    'mdm_profiles': ['lukeheath', 'zwass'],//« for dogfood.fleetdm.com, this is the required OS settings applied to contributor Macs
    'vpn': ['rfairburn', 'zwass'],// « for managing VPN rules for accessing customer and Fleet Sandbox infrastructure
    '.github/workflows': ['mikermcneil', 'zwass', 'hollidayn', 'lukeheath'],//« CI/CD workflows

    // Repo automation and change control settings
    'CODEOWNERS': ['mikermcneil', 'zwass'],
    '.gitignore': ['mikermcneil', 'zwass', 'hollidayn', 'dherder', 'zayhanlon', 'lukeheath', 'rfairburn'],// « what files should not be checked in?
    'free-for-all': '*',//« Folder that any fleetie (core team member, not consultants) can push to, willy-nilly

    // "Secret handbook"
    // Standard operating procedures (SOP), etc that would be public handbook content except for that it's confidential.
    'README.md': ['mikermcneil'],// « about this repo
    'cold-outbound-strategy.md': ['mikermcneil','bradmacd'],// « Cold outbound strategy (see fleetdm.com/handbook/company/why-this-way for our vision of a better way to sell)

    // GitHub issue templates
    '.github/ISSUE_TEMPLATE': ['mikermcneil'],// FUTURE: Bust out individual maintainership for issue templates once relevant DRIs are GitHub, markdown, and content design-certified

  },

  fleetMdmGitopsGithubRepoMaintainersByPath: {
    '/': ['lukeheath']
  },

  // The version of osquery to use when generating schema docs
  // (both in Fleet's query console and on fleetdm.com)
  versionOfOsquerySchemaToUseWhenGeneratingDocumentation: '5.8.1',

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
  // datadogApiKey: '…',

  // For receive-from-customer-fleet-instance webhook.
  // customerWorkspaceOneBaseUrl: '…',
  // customerWorkspaceOneOauthId: '…',
  // customerWorkspaceOneOauthSecret: '…',
  // customerMigrationWebhookSecret: '…',

  //…

};

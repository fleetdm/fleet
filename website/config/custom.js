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


  // Other integrations:
  // openAiSecret: undefined,
  // iqSecret: undefined, // You gotta use the base64-encoded API secret.  (Get it in your account settings in LeadIQ.)
  // salesforceIntegrationUsername: undefined,
  // salesforceIntegrationPasskey: undefined,

  // For cleaning up LinkedIn URLs before creating CRM records.
  RX_PROTOCOL_AND_COMMON_SUBDOMAINS: /^(https?\:\/\/)?(www\.|about\.|ch\.|uk\.|pl\.|ca\.|jp\.|im\.|fr\.|pt\.|vn\.|pk\.|in\.|lu\.|mu\.|nl\.|np\.)*/,

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
    //    'articles': { dri: 'mike-j-thomas', maintainers: ['mike-j-thomas', 'mike-j-thomas', 'mikermcneil'], repo: 'fleetdm/fleet' },
    //    ````
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    // 🚀 Code for core product and integrations
    'ee/tools/puppet': 'georgekarrv', //« Puppet integration (especially useful with macOS MDM turned on) -- FYI: Originally developed by request from "customer-eponym"
    'tools/api': 'lukeheath', //« Scripts used to interact with the Fleet API

    // ⚗️ Reference, config surface, built-in queries, API, and other documentation
    // 'docs/Using-Fleet/REST-API.md': '',              // « Covered in CODEOWNERS (2023-07-22)
    // 'docs/Contributing/reference/api-for-contributors.md': '', // « Covered in CODEOWNERS (2023-07-22)
    // 'schema': '',                                    // « Covered in CODEOWNERS (2023-07-22)
    'docs/01-Using-Fleet/standard-query-library/standard-query-library.yml': 'rachaelshaw', //« Built-in queries
    '/docs/get-started/faq': 'zayhanlon',
    'ee/cis': 'sharon-fdm',//« Fleet Premium only: built-in queries  (built-in policies for CIS benchmarks)  -- FYI: On 2023-07-15, we changed this so that Sharon, Lucas, and Rachel are all maintainers, but where there is a single DRI who is automatically requested approval from.

    // Fleet's internal IT and security (+dogfooding)
    'it-and-security': 'allenhouchins',

    // 🫧 Articles and release notes
    // 'articles': 'mike-j-thomas', « Covered in CODEOWNERS
    'CHANGELOG.md': 'lukeheath',

    // 🫧 Website (fleetdm.com)
    'website': 'eashaw',// (catch-all)
    'website/assets': 'eashaw', // « Eric is DRI for website frontend code
    'website/views': 'eashaw',
    'website/api': 'eashaw',//« Website backend, scripts, deps
    'website/api/controllers/webhooks/receive-from-github.js': 'eashaw',// github bot (webhook)
    'website/config': 'eashaw',
    'website/config/routes.js': 'eashaw',//« Website redirects and URLs
    'website/scripts': 'eashaw',
    'website/package.json': 'eashaw',

    // 🫧 Vulnerability dashboard
    'ee/vulnerability-dashboard': 'eashaw',// (catch-all)
    'ee/vulnerability-dashboard/assets': 'eashaw',
    'ee/vulnerability-dashboard/views': 'eashaw',
    'ee/vulnerability-dashboard/api': 'eashaw',//« Vulnerability dashboard backend, scripts, deps
    'ee/vulnerability-dashboard/config': 'eashaw',
    'ee/vulnerability-dashboard/config/routes.js': 'eashaw',//« Vulnerability dashboard redirects and URLs
    'ee/vulnerability-dashboard/scripts': 'eashaw',
    'ee/vulnerability-dashboard/package.json': 'eashaw',

    // 🫧 Bulk operations dashboard
    'ee/bulk-operations-dashboard': 'eashaw',// (catch-all)

    // 🫧 Pricing and features
    'handbook/company/pricing-features-table.yml': 'noahtalerman',
    'handbook/company/testimonials.yml': 'mike-j-thomas',

    // Dev process
    'handbook/company/product-groups.md': 'lukeheath',
    'handbook/engineering': 'lukeheath',
    'handbook/product-design': 'noahtalerman',


    // 🫧 Other brandfronts
    'README.md': 'mikermcneil',// « GitHub brandfront
    'tools/fleetctl-npm/README.md': 'mikermcneil',// « NPM brandfront (npmjs.com/package/fleetctl)

    // 🌐 Repo automation and change control settings
    'CODEOWNERS': 'sampfluger88',
    'website/config/custom.js': 'sampfluger88',
    '.gitignore': 'sampfluger88',// « what files should not be checked in?

    // 🌐 Handbook
    //'handbook': 'samfp', Covered in CODEOWNERS (#16972 2024-02-19)


    // 🌐 GitHub issue templates
    '.github/ISSUE_TEMPLATE': 'sampfluger88',

    // 💝 Fleet-maintained apps
    'ee/maintained-apps/inputs': 'mostlikelee',
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
    'ee/tools/puppet': ['lukeheath', 'gillespi314', 'mna', 'georgekarrv'],
    'tools/api': ['lukeheath', 'georgekarrv', 'sharon-fdm'],//« Scripts for interacting with the Fleet API

    // Reference, config surface, built-in queries, API, and other documentation
    'docs': ['rachaelshaw', 'noahtalerman', 'eashaw'],// (default for docs)
    'docs/01-Using-Fleet/standard-query-library/standard-query-library.yml': ['rachaelshaw', 'noahtalerman', 'eashaw'],// (standard query library)
    '/docs/get-started/faq': ['ksatter', 'ddribeiro', 'zayhanlon'],
    'docs/REST API/rest-api.md': ['rachaelshaw', 'lukeheath'],// (standard query library)
    'schema': ['eashaw'],// (Osquery table schema)
    'ee/cis': ['lukeheath', 'sharon-fdm', 'lucasmrod', 'rachelElysia', 'rachaelshaw'],

    // Fleet's internal IT and security (+dogfooding)
    'it-and-security': ['allenhouchins'],

    // Articles and release notes
    'CHANGELOG.md': ['mikermcneil', 'noahtalerman', 'lukeheath'],
    'articles': ['mike-j-thomas', 'eashaw', 'mikermcneil', 'rachaelshaw', 'lukeheath'],
    'website/assets/images/articles': ['mike-j-thomas', 'eashaw', 'mikermcneil'],

    // Website (fleetdm.com)
    'website': ['mikermcneil', 'eashaw'],// (default for website)
    'website/views': ['eashaw', 'mike-j-thomas'],
    'website/generators': 'eashaw',
    'website/assets': 'eashaw',
    'website/package.json': 'eashaw',
    'website/config/routes.js': ['eashaw', 'mike-j-thomas'],// (for managing website URLs)
    'website/config/policies.js': ['eashaw', 'mikermcneil'],// (for adding new pages and managing permissions)

    // 🫧 Vulnerability dashboard
    'ee/vulnerability-dashboard': ['eashaw', 'mikermcneil'],// (catch-all)
    'ee/vulnerability-dashboard/assets': 'eashaw',
    'ee/vulnerability-dashboard/views': 'eashaw',
    'ee/vulnerability-dashboard/config/routes.js': 'eashaw',
    'ee/vulnerability-dashboard/package.json': 'eashaw',

    // 🫧 Bulk operations dashboard
    'ee/bulk-operations-dashboard': 'eashaw',

    // Other brandfronts
    'README.md': ['mikermcneil', 'mike-j-thomas', 'lukeheath'],//« github brandfront (github.com/fleetdm/fleet)
    'tools/fleetctl-npm/README.md': ['mikermcneil', 'mike-j-thomas', 'lukeheath'],//« brandfront for fleetctl package on npm (npmjs.com/package/fleetctl)

    // Config as code for infrastructure, internal security and IT use cases, and more.
    //'infrastructure': [],// Decided against in https://github.com/fleetdm/fleet/pull/12890
    //'charts': [], //Decided against in https://github.com/fleetdm/fleet/pull/12890
    //'terraform': [],//Decided against in https://github.com/fleetdm/fleet/pull/12890

    // Github workflows
    '.github/workflows/deploy-fleet-website.yml': ['eashaw','mikermcneil'],// (website deploy script)
    '.github/workflows/test-website.yml': ['eashaw','mikermcneil'],//« website CI test script
    '.github/workflows/deploy-vulnerability-dashboard.yml': ['eashaw','mikermcneil'],// (vulnerabiltiy dashboard deploy script)
    '.github/workflows/test-vulnerability-dashboard-changes.yml': ['eashaw','mikermcneil'],//« vulnerabiltiy dashboard CI test script
    '.github/workflows': ['lukeheath', 'mikermcneil'],//« CI/CD workflows & misc GitHub Actions. Note that some are also addressed more specifically below in relevant sections)

    // Repo automation and change control settings
    'CODEOWNERS': ['mikermcneil', 'sampfluger88', 'lukeheath'],// (« for changing who reviews is automatically requested from for given paths)
    'website/config/custom.js': ['eashaw', 'mikermcneil', 'lukeheath', 'sampfluger88'],// (« for changing whose changes automatically approve and unfreeze relevant PRs changing given paths)

    // Handbook
    'handbook/README.md': 'mikermcneil', // See https://github.com/fleetdm/fleet/pull/13195
    'handbook/company': 'mikermcneil',
    'handbook/company/open-positions.yml': ['sampfluger88', 'mikermcneil'],
    'handbook/company/communications.md': ['sampfluger88', 'mikermcneil'],
    'handbook/company/go-to-market-groups.md': ['sampfluger88', 'mikermcneil'],
    'handbook/company/leadership.md': ['sampfluger88', 'mikermcneil'],
    'handbook/it-and-enablement': ['sampfluger88', 'mikermcneil'],
    'handbook/finance': ['sampfluger88', 'mikermcneil'],
    'handbook/sales': ['sampfluger88', 'mikermcneil'],
    'handbook/marketing': ['sampfluger88', 'mikermcneil'],
    'handbook/customer-success': ['sampfluger88', ' mikermcneil'],

    // 🫧 Pricing and features and dev process
    'handbook/company/pricing-features-table.yml': ['noahtalerman', 'mikermcneil'],
    'handbook/company/testimonials.yml': ['eashaw', 'mike-j-thomas', 'zayhanlon'],

    // Dev process
    'handbook/company/product-groups.md': ['lukeheath', 'noahtalerman', 'sampfluger88', 'mikermcneil'],
    'handbook/engineering': ['sampfluger88', 'lukeheath'],
    'handbook/product-design': ['sampfluger88', 'noahtalerman'],

    // GitHub issue templates
    '.github/ISSUE_TEMPLATE': ['mikermcneil', 'sampfluger88'],
    '.github/ISSUE_TEMPLATE/bug-report.md': ['lukeheath', 'xpkoala','noahtalerman'],
    '.github/ISSUE_TEMPLATE/feature-request.md': ['lukeheath', 'xpkoala', 'noahtalerman'],
    '.github/ISSUE_TEMPLATE/release-qa.md': ['lukeheath', 'xpkoala', 'noahtalerman'],
  },

  confidentialGithubRepoMaintainersByPath: {// fleetdm/confidential

    // Config as code for infrastructure, internal security and IT use cases, and more.
    'mdm_profiles': ['lukeheath'],//« for dogfood.fleetdm.com, this is the required OS settings applied to contributor Macs
    'vpn': ['rfairburn', 'lukeheath'],// « for managing VPN rules for accessing customer and Fleet Sandbox infrastructure
    '.github/workflows': ['sampfluger88', 'lukeheath'],//« CI/CD workflows

    // Repo automation and change control settings
    'CODEOWNERS': ['mikermcneil', 'sampfluger88', 'lukeheath'],
    '.gitignore': ['lukeheath', 'rfairburn', 'sampfluger88'],// « what files should not be checked in?
    'free-for-all': '*',//« Folder that any fleetie (core team member, not consultants) can push to, willy-nilly

    // "Secret handbook"
    // Standard operating procedures (SOP), etc that would be public handbook content except for that it's confidential.
    'README.md': ['mikermcneil'],// « about this repo

  },

  fleetMdmGitopsGithubRepoMaintainersByPath: {
    '/': ['lukeheath'] // Future update this
  },

  //  ███████╗ ██████╗██╗  ██╗███████╗███╗   ███╗ █████╗
  //  ██╔════╝██╔════╝██║  ██║██╔════╝████╗ ████║██╔══██╗
  //  ███████╗██║     ███████║█████╗  ██╔████╔██║███████║
  //  ╚════██║██║     ██╔══██║██╔══╝  ██║╚██╔╝██║██╔══██║
  //  ███████║╚██████╗██║  ██║███████╗██║ ╚═╝ ██║██║  ██║
  //  ╚══════╝ ╚═════╝╚═╝  ╚═╝╚══════╝╚═╝     ╚═╝╚═╝  ╚═╝
  //
  // The version of osquery to use when generating schema docs
  // (both in Fleet's query console and on fleetdm.com)
  versionOfOsquerySchemaToUseWhenGeneratingDocumentation: '5.18.1',


  //  ███╗   ███╗██╗███████╗ ██████╗
  //  ████╗ ████║██║██╔════╝██╔════╝
  //  ██╔████╔██║██║███████╗██║
  //  ██║╚██╔╝██║██║╚════██║██║
  //  ██║ ╚═╝ ██║██║███████║╚██████╗
  //  ╚═╝     ╚═╝╚═╝╚══════╝ ╚═════╝
  //
  /***************************************************************************
  *                                                                          *
  * Any other custom config this Sails app should use during development.    *
  * (and possibly in ALL environments, if not overridden in config/env/)     *
  *                                                                          *
  ***************************************************************************/

  // FUTURE: Consolidate these two lists of email domains (And maybe find another word for banned)
  // For the deliver-apple-csr webhook:
  bannedEmailDomainsForCSRSigning:   [
    'aim.com',         'alice.it',     'aliceadsl.fr',     'aol.com',
    'arcor.de',        'att.net',      'bellsouth.net',    'bigpond.com',
    'bigpond.net.au',  'bluewin.ch',   'blueyonder.co.uk', 'bol.com.br',
    'centurytel.net',  'charter.net',  'chello.nl',        'club-internet.fr',
    'comcast.net',     'cox.net',      'earthlink.net',    'facebook.com',
    'free.fr',         'freenet.de',   'frontiernet.net',  'gmail.com',
    'gmx.de',          'gmx.net',      'googlemail.com',   'hetnet.nl',
    'home.nl',         'hotmail.ca',   'hotmail.co.uk',    'hotmail.com',
    'hotmail.de',      'hotmail.es',   'hotmail.fr',       'hotmail.it',
    'icloud.com',      'ig.com.br',    'juno.com',         'laposte.net',
    'libero.it',       'live.ca',      'live.co.uk',       'live.com',
    'live.com.au',     'live.fr',      'live.it',          'live.nl',
    'mac.com',         'mail.com',     'mail.ru',          'me.com',
    'msn.com',         'neuf.fr',      'ntlworld.com',     'optonline.net',
    'optusnet.com.au', 'orange.fr',    'outlook.com',      'planet.nl',
    'pm.me',           'proton.me',    'protonmail.ch',    'protonmail.com',
    'qq.com',          'rambler.ru',   'rediffmail.com',   'rocketmail.com',
    'sbcglobal.net',   'sfr.fr',       'shaw.ca',          'sky.com',
    'skynet.be',       'sympatico.ca', 't-online.de',      'telenet.be',
    'terra.com.br',    'tin.it',       'tiscali.co.uk',    'tiscali.it',
    'tmmbt.net',       'uol.com.br',   'verizon.net',      'virgilio.it',
    'voila.fr',        'wanadoo.fr',   'web.de',           'windstream.net',
    'yahoo.ca',        'yahoo.co.id',  'yahoo.co.in',      'yahoo.co.jp',
    'yahoo.co.uk',     'yahoo.com',    'yahoo.com.ar',     'yahoo.com.au',
    'yahoo.com.br',    'yahoo.com.mx', 'yahoo.com.sg',     'yahoo.de',
    'yahoo.es',        'yahoo.fr',     'yahoo.in',         'yahoo.it',
    'yandex.ru',       'ymail.com',    'zoho.com',         'zonnet.nl'
  ],

  // For website signups & contact form submissions:
  bannedEmailDomainsForWebsiteSubmissions: [
    'gmail.com',
    'yahoo.com',
    'yahoo.co.uk',
    'hotmail.com',
    'hotmail.co.uk',
    'hotmail.ca',
    'outlook.com',
    'icloud.com',
    'proton.me',
    'live.com',
    'yandex.ru',
    'ymail.com',
    'qq.com',
  ],

  // Zapier:
  // zapierWebhookSecret: '…',

  // Contact form:
  // slackWebhookUrlForContactForm: '…',
  // slackWebhookUrlForNewlyCreatedOppts: '…',

  // GitHub bot:
  // githubAccessToken: '…',
  // githubBotWebhookSecret: '…',
  // githubProjectsV2ItemWebhookSecret: '…',
  // slackWebhookUrlForGithubBot: '…',
  // mergeFreezeAccessToken: '…',

  // Metrics:
  // engMetricsGcpServiceAccountKey: '…',
  // datadogApiKey: '…',

  // For receive-from-customer-fleet-instance webhook.
  // customerWorkspaceOneBaseUrl: '…',
  // customerWorkspaceOneOauthId: '…',
  // customerWorkspaceOneOauthSecret: '…',
  // customerMigrationWebhookSecret: '…',

  // For nurture emails:
  // contactEmailForNutureEmails: '…',
  // activityCaptureEmailForNutureEmails: '…',
  // contactNameForNurtureEmails: '…',

  // Deal registration form
  // dealRegistrationContactEmailAddress: '…',


  // Microsoft compliance proxy
  // compliancePartnerClientId: '…',
  // compliancePartnerClientSecret: '…',
  // cloudCustomerCompliancePartnerSharedSecret: '…',


  // Android proxy
  // androidEnterpriseProjectId: '…',
  // androidEnterpriseServiceAccountEmailAddress: '…',
  // androidEnterpriseServiceAccountPrivateKey: '…',


};

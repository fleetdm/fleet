/**
 * Route Mappings
 * (sails.config.routes)
 *
 * Your routes tell Sails what to do each time it receives a request.
 *
 * For more information on configuring custom routes, check out:
 * https://sailsjs.com/anatomy/config/routes-js
 */

module.exports.routes = {

  //  ╦ ╦╔═╗╔╗ ╔═╗╔═╗╔═╗╔═╗╔═╗
  //  ║║║║╣ ╠╩╗╠═╝╠═╣║ ╦║╣ ╚═╗
  //  ╚╩╝╚═╝╚═╝╩  ╩ ╩╚═╝╚═╝╚═╝
  'GET /': {
    action: 'view-homepage-or-redirect',
    locals: { isHomepage: true }
  },

  'GET /company/contact': {
    action: 'view-contact',
    locals: {
      pageTitleForMeta: 'Contact us | Fleet for osquery',
      pageDescriptionForMeta: 'Get in touch with our team.'
    }
  },

  'GET /fleetctl-preview': {
    action: 'view-get-started',
    locals: {
      pageTitleForMeta: 'fleetctl preview | Fleet for osquery',
      pageDescriptionForMeta: 'Learn about getting started with Fleet using fleetctl.'
    }
  },

  'GET /pricing': {
    action: 'view-pricing',
    locals: {
      currentSection: 'pricing',
      pageTitleForMeta: 'Pricing | Fleet for osquery',
      pageDescriptionForMeta: 'View Fleet plans and pricing details.'
    }
  },

  'GET /logos': {
    action: 'view-press-kit',
    locals: {
      pageTitleForMeta: 'Logos | Fleet for osquery',
      pageDescriptionForMeta: 'Download Fleet logos, wallpapers, and screenshots.'
    }
  },

  'GET /queries': {
    action: 'view-query-library',
    locals: {
      currentSection: 'documentation',
      pageTitleForMeta: 'Queries | Fleet for osquery',
      pageDescriptionForMeta: 'A growing collection of useful queries for organizations deploying Fleet and osquery.'
    }
  },

  'GET /queries/:slug': {
    action: 'view-query-detail',
    locals: {
      currentSection: 'documentation',
    }
  },

  'r|^/((success-stories|securing|releases|engineering|guides|announcements|podcasts|report|deploy)/(.+))$|': {
    skipAssets: false,
    action: 'articles/view-basic-article',
  },// Handles /device-management/foo, /securing/foo, /releases/foo, /engineering/foo, /guides/foo, /announcements/foo, /deploy/foo, /podcasts/foo, /report/foo

  'r|^/((success-stories|securing|releases|engineering|guides|announcements|articles|podcasts|report|deploy))/*$|category': {
    skipAssets: false,
    action: 'articles/view-articles',
  },// Handles the article landing page /articles, and the article cateogry pages (e.g. /device-management, /securing, /releases, etc)

  'GET /docs/?*': {
    skipAssets: false,
    action: 'docs/view-basic-documentation',
    locals: {
      currentSection: 'documentation',
    }
  },// handles /docs and /docs/foo/bar

  'GET /handbook/?*':  {
    skipAssets: false,
    action: 'handbook/view-basic-handbook',
    locals: {
      currentSection: 'community',
    }
  },// handles /handbook and /handbook/foo/bar

  'GET /transparency': {
    action: 'view-transparency',
    locals: {
      pageTitleForMeta: 'Transparency | Fleet for osquery',
      pageDescriptionForMeta: 'Learn what data osquery can see.',
    }
  },
  'GET /customers/new-license': {
    action: 'customers/view-new-license',
    locals: {
      layout: 'layouts/layout-customer',
      pageTitleForMeta: 'Get Fleet Premium | Fleet for osquery',
      pageDescriptionForMeta: 'Generate your quote and start using Fleet Premium today.',
    }
  },
  'GET /customers/register': {
    action: 'entrance/view-signup',
    locals: {
      layout: 'layouts/layout-customer',
      pageTitleForMeta: 'Sign up | Fleet for osquery',
      pageDescriptionForMeta: 'Sign up for a Fleet Premium license.',
    }
  },
  'GET /customers/login': {
    action: 'entrance/view-login',
    locals: {
      layout: 'layouts/layout-customer',
      pageTitleForMeta: 'Log in | Fleet for osquery',
      pageDescriptionForMeta: 'Log in to the Fleet customer portal.',
    }
  },
  'GET /customers/dashboard': {
    action: 'customers/view-dashboard',
    locals: {
      layout: 'layouts/layout-customer',
      pageTitleForMeta: 'Customer dashboard | Fleet for osquery',
      pageDescriptionForMeta: 'View and edit information about your Fleet Premium license.',
    }
  },
  'GET /customers/forgot-password': {
    action: 'entrance/view-forgot-password',
    locals: {
      layout: 'layouts/layout-customer',
      pageTitleForMeta: 'Forgot password | Fleet for osquery',
      pageDescriptionForMeta: 'Recover the password for your Fleet customer account.',
    }
  },
  'GET /customers/new-password': {
    action: 'entrance/view-new-password',
    locals: {
      layout: 'layouts/layout-customer',
      pageTitleForMeta: 'New password | Fleet for osquery',
      pageDescriptionForMeta: 'Change the password for your Fleet customer account.',
    }
  },

  'GET /reports/state-of-device-management': {
    action: 'reports/view-state-of-device-management',
    locals: {
      pageTitleForMeta: 'State of device management | Fleet for osquery',
      pageDescriptionForMeta: 'We surveyed 200+ security practitioners to discover the state of device management in 2022. Click here to learn about their struggles and best practices.',
      headerCTAHidden: true,
    }
  },

  'GET /overview': {
    action: 'view-sales-one-pager',
    locals: {
      pageTitleForMeta: 'Overview | Fleet for osquery',
      pageDescriptionForMeta: 'Fleet helps security and IT teams protect their devices. We\'re the single source of truth for workstation and server telemetry. Click to learn more!',
      layout: 'layouts/layout-landing'
    },
  },

  'GET /try-fleet/register': {
    action: 'try-fleet/view-register',
    locals: {
      layout: 'layouts/layout-sandbox',
      pageTitleForMeta: 'Fleet Sandbox | Fleet for osquery',
      pageDescriptionForMeta: 'Fleet Sandbox - The fastest way to test Fleet. Get up and running in minutes to try out Fleet.',
    }
  },

  'GET /try-fleet/login': {
    action: 'try-fleet/view-sandbox-login',
    locals: {
      layout: 'layouts/layout-sandbox',
      pageTitleForMeta: 'Log in to Fleet Sandbox | Fleet for osquery',
      pageDescriptionForMeta: 'Log in to Fleet Sandbox.',
    }
  },

  'GET /try-fleet/sandbox': {
    action: 'try-fleet/view-sandbox-teleporter-or-redirect-because-expired-or-waitlist',
    locals: {
      layout: 'layouts/layout-sandbox',
    },
  },

  'GET /try-fleet/sandbox-expired': {
    action: 'try-fleet/view-sandbox-expired',
    locals: {
      layout: 'layouts/layout-sandbox',
    },
  },

  'GET /try-fleet/waitlist': {
    action: 'try-fleet/view-waitlist',
    locals: {
      layout: 'layouts/layout-sandbox',
      pageTitleForMeta: 'Fleet Sandbox waitlist | Fleet for osquery',
    }
  },

  'GET /admin/email-preview': {
    action: 'admin/view-email-templates',
    locals: {
      layout: 'layouts/layout-customer'
    },
  },

  'GET /admin/email-preview/*': {
    action: 'admin/view-email-template-preview',
    skipAssets: true,
    locals: {
      layout: 'layouts/layout-customer'
    },
  },

  'GET /tables/:tableName': {
    action: 'view-osquery-table-details',
    locals: {
      currentSection: 'documentation',
    }
  },

  'GET /admin/generate-license': {
    action: 'admin/view-generate-license',
    locals: {
      layout: 'layouts/layout-customer'
    }
  },


  'GET /connect-vanta': {
    action: 'view-connect-vanta',
    locals: {
      layout: 'layouts/layout-sandbox',
    }
  },

  'GET /vanta-authorization': {
    action: 'view-vanta-authorization',
    locals: {
      layout: 'layouts/layout-sandbox',
    }
  },

  'GET /device-management': {
    action: 'view-fleet-mdm',
    locals: {
      pageTitleForMeta: 'Device management | Fleet for osquery',
      pageDescriptionForMeta: 'GitOps-driven MDM. Automate the management of your fleet of devices with increased visibility, control, and improved stability.',
      currentSection: 'platform',
    }
  },

  'GET /upgrade': {
    action: 'view-upgrade',
    locals: {
      pageTitleForMeta: 'Upgrade to Fleet Premium | Fleet for osquery',
      pageDescriptionForMeta: 'Learn about the benefits of upgrading to Fleet Premium',
    }
  },

  'GET /compliance': {
    action: 'view-compliance',
    locals: {
      currentSection: 'platform',
      pageTitleForMeta: 'Security compliance | Fleet for osquery',
      pageDescriptionForMeta: 'Automate security workflows by creating or installing policies to maintain your organization\'s compliance goals. Simplify security compliance with Fleet.',
    }
  },

  'GET /osquery-management': {
    action: 'view-osquery-management',
    locals: {
      pageTitleForMeta: 'Osquery management | Fleet for osquery',
      pageDescriptionForMeta: 'Fleet lets you harness the power of osquery to stream accurate, real-time data from all your endpoints.',
    }
  },

  'GET /vulnerability-management': {
    action: 'view-vulnerability-management',
    locals: {
      pageTitleForMeta: 'Vulnerability management | Fleet for osquery',
      pageDescriptionForMeta: 'Know what\’s going on with your computers. Measure and automate risk across your laptops and servers.',
    }
  },

  'GET /support': {
    action: 'view-support',
    locals: {
      pageTitleForMeta: 'Support | Fleet for osquery',
      pageDescriptionForMeta: 'Ask a question, chat with other engineers, or get in touch with the Fleet team.',
      currentSection: 'documentation',
    }
  },


  //  ╦╔╦╗╔═╗╔═╗╦╔╗╔╔═╗  ┌─┬  ┌─┐┌┐┌┌┬┐┬┌┐┌┌─┐  ┌─┐┌─┐┌─┐┌─┐┌─┐─┐
  //  ║║║║╠═╣║ ╦║║║║║╣   │ │  ├─┤│││ │││││││ ┬  ├─┘├─┤│ ┬├┤ └─┐ │
  //  ╩╩ ╩╩ ╩╚═╝╩╝╚╝╚═╝  └─┴─┘┴ ┴┘└┘─┴┘┴┘└┘└─┘  ┴  ┴ ┴└─┘└─┘└─┘─┘
  'GET /imagine/unused-software': { action: 'imagine/view-unused-software' },
  'GET /imagine/higher-education': {
    action: 'imagine/view-higher-education',
    locals: {
      pageTitleForMeta: 'Fleet for higher education',
      pageDescriptionForMeta: 'Automate security workflows in a single application by creating or installing policies to identify which devices comply with your security guidelines.',
    }
  },
  'GET /imagine/rapid-7-alternative': {
    action: 'imagine/view-rapid-7-alternative',
    locals: {
      pageTitleForMeta: 'An open-source alternative to Rapid7',
      pageDescriptionForMeta: 'Simplify vulnerability management with Fleet, an open-source platform with superior visibility.',
    }
  },
  'GET /imagine/defcon-31': {
    action: 'imagine/view-defcon-31',
    locals: {
      pageTitleForMeta: 'Fleet at DefCon 31',
      pageDescriptionForMeta: 'Find Fleet at DefCon and get a custom tee shirt.',
    }
  },

  //  ╦  ╔═╗╔═╗╔═╗╔═╗╦ ╦  ╦═╗╔═╗╔╦╗╦╦═╗╔═╗╔═╗╔╦╗╔═╗
  //  ║  ║╣ ║ ╦╠═╣║  ╚╦╝  ╠╦╝║╣  ║║║╠╦╝║╣ ║   ║ ╚═╗
  //  ╩═╝╚═╝╚═╝╩ ╩╚═╝ ╩   ╩╚═╚═╝═╩╝╩╩╚═╚═╝╚═╝ ╩ ╚═╝
  //  ┌─  ┌─┐┌─┐┬─┐  ┌┐ ┌─┐┌─┐┬┌─┬ ┬┌─┐┬─┐┌┬┐┌─┐  ┌─┐┌─┐┌┬┐┌─┐┌─┐┌┬┐  ─┐
  //  │   ├┤ │ │├┬┘  ├┴┐├─┤│  ├┴┐│││├─┤├┬┘ ││└─┐  │  │ ││││├─┘├─┤ │    │
  //  └─  └  └─┘┴└─  └─┘┴ ┴└─┘┴ ┴└┴┘┴ ┴┴└──┴┘└─┘  └─┘└─┘┴ ┴┴  ┴ ┴ ┴o  ─┘
  // Add redirects here for deprecated/legacy links, so that they go to an appropriate new place instead of just being broken when pages move or get renamed.
  //
  // For example:
  // If we were going to change fleetdm.com/company/about to fleetdm.com/company/story, we might do something like:
  // ```
  // 'GET /company/about': '/company/story',
  // ```
  //
  // Or another example, if we were to rename a doc page:
  // ```
  // 'GET /docs/using-fleet/learn-how-to-use-fleet': '/docs/using-fleet/fleet-for-beginners',
  // ```
  'GET /try-fleet':                  '/get-started',
  'GET /try': '/get-started',
  'GET /docs/deploying/fleet-public-load-testing': '/docs/deploying/load-testing',
  'GET /handbook/customer-experience': '/handbook/customers',
  'GET /handbook/brand': '/handbook/digital-experience',
  'GET /guides/deploying-fleet-on-aws-with-terraform': '/deploy/deploying-fleet-on-aws-with-terraform',
  'GET /guides/deploy-fleet-on-hetzner-cloud':'/deploy/deploy-fleet-on-hetzner-cloud',
  'GET /guides/deploying-fleet-on-render': '/deploy/deploying-fleet-on-render',
  'GET /use-cases/correlate-network-connections-with-community-id-in-osquery': '/guides/correlate-network-connections-with-community-id-in-osquery',
  'GET /use-cases/converting-unix-timestamps-with-osquery': '/guides/converting-unix-timestamps-with-osquery',
  'GET /use-cases/ebpf-the-future-of-osquery-on-linux': '/securing/ebpf-the-future-of-osquery-on-linux',
  'GET /use-cases/fleet-quick-tips-querying-procdump-eula-has-been-accepted': '/guides/fleet-quick-tips-querying-procdump-eula-has-been-accepted',
  'GET /use-cases/generate-process-trees-with-osquery': '/guides/generate-process-trees-with-osquery',
  'GET /use-cases/get-and-stay-compliant-across-your-devices-with-fleet': '/securing/get-and-stay-compliant-across-your-devices-with-fleet',
  'GET /use-cases/import-and-export-queries-and-packs-in-fleet': '/guides/import-and-export-queries-and-packs-in-fleet',
  'GET /guides/import-and-export-queries-and-packs-in-fleet': '/guides/import-and-export-queries-in-fleet',
  'GET /use-cases/locate-assets-with-osquery': '/guides/locate-assets-with-osquery',
  'GET /use-cases/osquery-a-tool-to-easily-ask-questions-about-operating-systems': '/guides/osquery-a-tool-to-easily-ask-questions-about-operating-systems',
  'GET /use-cases/osquery-consider-joining-against-the-users-table': '/guides/osquery-consider-joining-against-the-users-table',
  'GET /use-cases/stay-on-course-with-your-security-compliance-goals': '/guides/stay-on-course-with-your-security-compliance-goals',
  'GET /use-cases/using-elasticsearch-and-kibana-to-visualize-osquery-performance': '/guides/using-elasticsearch-and-kibana-to-visualize-osquery-performance',
  'GET /use-cases/work-may-be-watching-but-it-might-not-be-as-bad-as-you-think': '/securing/work-may-be-watching-but-it-might-not-be-as-bad-as-you-think',
  'GET /docs/contributing/testing':  '/docs/contributing/testing-and-local-development',
  'GET /handbook/sales': '/handbook/customers#sales',
  'GET /handbook/people': '/handbook/business-operations',
  'GET /handbook/people/ceo-handbook': '/handbook/business-operations/ceo-handbook',
  'GET /handbook/growth': '/handbook/marketing#growth',
  'GET /handbook/community': '/handbook/marketing#community',
  'GET /handbook/digital-experience': '/handbook/marketing#digital-experience',
  'GET /handbook/digital-experience/article-formatting-guide': '/handbook/marketing/article-formatting-guide',
  'GET /handbook/digital-experience/commonly-used-terms': '/handbook/marketing/commonly-used-terms',
  'GET /handbook/digital-experience/how-to-submit-and-publish-an-article': '/handbook/marketing/how-to-submit-and-publish-an-article',
  'GET /handbook/digital-experience/markdown-guide': '/handbook/marketing/markdown-guide',
  'GET /handbook/quality': '/handbook/engineering#quality',
  'GET /device-management/fleet-user-stories-f100': '/success-stories/fleet-user-stories-wayfair',
  'GET /device-management/fleet-user-stories-schrodinger': '/success-stories/fleet-user-stories-wayfair',
  'GET /device-management/fleet-user-stories-wayfair': '/success-stories/fleet-user-stories-wayfair',
  'GET /handbook/security': '/handbook/business-operations/security',
  'GET /handbook/security/security-policies':'/handbook/business-operations/security-policies#information-security-policy-and-acceptable-use-policy',// « reasoning: https://github.com/fleetdm/fleet/pull/9624
  'GET /handbook/handbook': '/handbook/company/handbook',
  'GET /handbook/company/development-groups': '/handbook/company/product-groups',
  'GET /docs/using-fleet/mdm-macos-settings': '/docs/using-fleet/mdm-custom-macos-settings',
  'GET /platform': '/',
  'GET /handbook/company/senior-software-backend-engineer': 'https://www.linkedin.com/posts/mikermcneil_in-addition-to-our-product-quality-specialist-activity-7067711903166279680-6CMH',
  'GET /handbook/business-operations/ceo-handbook': '/handbook/company/ceo-handbook',

  'GET /docs': '/docs/get-started/why-fleet',
  'GET /docs/get-started': '/docs/get-started/why-fleet',
  'GET /docs/rest-api': '/docs/rest-api/rest-api',
  'GET /docs/using-fleet': '/docs/using-fleet/fleet-ui',
  'GET /docs/configuration': '/docs/configuration/fleet-server-configuration',
  'GET /docs/contributing': 'https://github.com/fleetdm/fleet/tree/main/docs/Contributing',
  'GET /docs/deploy': '/docs/deploy/introduction',
  'GET /docs/using-fleet/faq': '/docs/get-started/faq',
  'GET /docs/using-fleet/monitoring-fleet': '/docs/deploy/monitoring-fleet',
  'GET /docs/using-fleet/adding-hosts': '/docs/using-fleet/enroll-hosts',
  'GET /docs/using-fleet/teams': '/docs/using-fleet/segment-hosts',
  'GET /docs/using-fleet/permissions': '/docs/using-fleet/manage-access',
  'GET /docs/using-fleet/chromeos': '/docs/using-fleet/enroll-chromebooks',
  'GET /docs/using-fleet/rest-api': '/docs/rest-api/rest-api',
  'GET /docs/using-fleet/configuration-files': '/docs/configuration/configuration-files/',
  'GET /docs/using-fleet/application-security': '/handbook/business-operations/application-security',
  'GET /docs/using-fleet/security-audits': '/handbook/business-operations/security-audits',
  'GET /docs/using-fleet/process-file-events': '/guides/querying-process-file-events-table-on-centos-7',
  'GET /docs/using-fleet/audit-activities': '/docs/using-fleet/audit-logs',
  'GET /docs/using-fleet/detail-queries-summary': '/docs/using-fleet/understanding-host-vitals',
  'GET /docs/using-fleet/orbit': '/docs/using-fleet/fleetd',
  'GET /docs/deploying': '/docs/deploy',
  'GET /docs/deploying/faq': '/docs/get-started/faq',
  'GET /docs/deploying/introduction': '/docs/deploy/introduction',
  'GET /docs/deploying/reference-architectures': '/docs/deploy/reference-architectures ',
  'GET /docs/deploying/upgrading-fleet': '/docs/deploy/upgrading-fleet',
  'GET /docs/deploying/server-installation': '/docs/deploy/server-installation',
  'GET /docs/deploying/cloudgov': '/docs/deploy/cloudgov',
  'GET /docs/deploying/configuration': '/docs/configuration/fleet-server-configuration',
  'GET /docs/deploying/fleetctl-agent-updates': '/docs/using-fleet/update-agents',
  'GET /docs/deploying/debugging': '/handbook/engineering/debugging',
  'GET /docs/deploying/load-testing': '/handbook/engineering/load-testing',
  'GET /docs/contributing/configuration': '/docs/configuration/configuration-files',
  'GET /docs/contributing/*': {
    skipAssets: true,
    fn: (req, res)=>{
      return res.redirect('https://github.com/fleetdm/fleet/tree/main/docs/Contributing');
    }
  },
  'GET /docs/contributing/orbit-development-and-release-strategy': '/docs/contributing/fleetd-development-and-release-strategy',
  'GET /docs/contributing/run-locally-built-orbit': '/docs/contributing/run-locally-built-fleetd',
  'GET /handbook/company/ceo-handbook': '/handbook/company/ceo',

  //  ╔╦╗╦╔═╗╔═╗  ╦═╗╔═╗╔╦╗╦╦═╗╔═╗╔═╗╔╦╗╔═╗   ┬   ╔╦╗╔═╗╦ ╦╔╗╔╦  ╔═╗╔═╗╔╦╗╔═╗
  //  ║║║║╚═╗║    ╠╦╝║╣  ║║║╠╦╝║╣ ║   ║ ╚═╗  ┌┼─   ║║║ ║║║║║║║║  ║ ║╠═╣ ║║╚═╗
  //  ╩ ╩╩╚═╝╚═╝  ╩╚═╚═╝═╩╝╩╩╚═╚═╝╚═╝ ╩ ╚═╝  └┘   ═╩╝╚═╝╚╩╝╝╚╝╩═╝╚═╝╩ ╩═╩╝╚═╝

  // Convenience
  // =============================================================================================================
  // Things that people are used to typing in to the URL and just randomly trying.
  //
  // For example, a clever user might try to visit fleetdm.com/documentation, not knowing that Fleet's website
  // puts this kind of thing under /docs, NOT /documentation.  These "convenience" redirects are to help them out.
  'GET /renew':                      'https://calendly.com/zayhanlon/fleet-renewal-discussion',
  'GET /documentation':              '/docs',
  'GET /contribute':                 '/docs/contributing',
  'GET /install':                    '/fleetctl-preview',
  'GET /company':                    '/company/about',
  'GET /company/about':              '/handbook', // FUTURE: brief "about" page explaining the origins of the company
  'GET /contact':                    '/company/contact',
  'GET /legal':                      '/legal/terms',
  'GET /terms':                      '/legal/terms',
  'GET /handbook/security/github':   '/handbook/security#git-hub-security',
  'GET /login':                      '/customers/login',
  'GET /slack':                      'https://join.slack.com/t/osquery/shared_invite/zt-1wkw5fzba-lWEyke60sjV6C4cdinFA1w',
  'GET /docs/using-fleet/updating-fleet': '/docs/deploying/upgrading-fleet',
  'GET /blog':                   '/articles',
  'GET /brand':                  '/logos',
  'GET /get-started':            '/try-fleet/register',
  'GET /g':                       (req,res)=> { let originalQueryStringWithAmp = req.url.match(/\?(.+)$/) ? '&'+req.url.match(/\?(.+)$/)[1] : ''; return res.redirect(301, sails.config.custom.baseUrl+'/?meet-fleet'+originalQueryStringWithAmp); },
  'GET /test-fleet-sandbox':     '/try-fleet/register',
  'GET /unsubscribe':             (req,res)=> { let originalQueryString = req.url.match(/\?(.+)$/) ? req.url.match(/\?(.+)$/)[1] : ''; return res.redirect(301, sails.config.custom.baseUrl+'/api/v1/unsubscribe-from-all-newsletters?'+originalQueryString);},
  'GET /tables':                 '/tables/account_policy_data',
  'GET /imagine/launch-party':   'https://www.eventbrite.com/e/601763519887',

  // Fleet UI
  // =============================================================================================================
  // These are external links not maintained by Fleet. We can point the Fleet UI to redirects here instead of the
  // original sources to help avoid broken links.
  'GET /learn-more-about/chromeos-updates': 'https://support.google.com/chrome/a/answer/6220366',

  // Sitemap
  // =============================================================================================================
  // This is for search engines, not humans.  Search engines know to visit fleetdm.com/sitemap.xml to download this
  // XML file, which helps search engines know which pages are available on the website.
  'GET /sitemap.xml':            { action: 'download-sitemap' },

  // RSS feeds
  // =============================================================================================================
  'GET /rss/:categoryName': {action: 'download-rss-feed'},

  // Potential future pages
  // =============================================================================================================
  // Things that are not webpages here (in the Sails app) yet, but could be in the future.  For now they are just
  // redirects to somewhere else EXTERNAL to the Sails app.
  'GET /security':               'https://github.com/fleetdm/fleet/security/policy',
  'GET /trust':                  'https://trust.fleetdm.com',
  'GET /status':                 'https://status.fleetdm.com',
  'GET /hall-of-fame':           'https://github.com/fleetdm/fleet/pulse',
  'GET /apply':                  '/jobs',
  'GET /jobs':                   'https://fleetdm.com/handbook/company#open-positions',
  'GET /company/stewardship':    'https://github.com/fleetdm/fleet', // FUTURE: page about how we approach open source and our commitments to the community
  'GET /legal/terms':            'https://docs.google.com/document/d/1OM6YDVIs7bP8wg6iA3VG13X086r64tWDqBSRudG4a0Y/edit',
  'GET /legal/privacy':          'https://docs.google.com/document/d/17i_g1aGpnuSmlqj35-yHJiwj7WRrLdC_Typc1Yb7aBE/edit',
  'GET /logout':                 '/api/v1/account/logout',
  'GET /defcon':                 'https://kqphpqst851.typeform.com/to/Y6NYxM5A',
  'GET /osquery-stickers':       'https://kqphpqst851.typeform.com/to/JxJ8YnxG',
  'GET /swag':                   'https://kqphpqst851.typeform.com/to/Y6NYxM5A',
  'GET /community':              'https://join.slack.com/t/osquery/shared_invite/zt-1wkw5fzba-lWEyke60sjV6C4cdinFA1w',


  //  ╦ ╦╔═╗╔╗ ╦ ╦╔═╗╔═╗╦╔═╔═╗
  //  ║║║║╣ ╠╩╗╠═╣║ ║║ ║╠╩╗╚═╗
  //  ╚╩╝╚═╝╚═╝╩ ╩╚═╝╚═╝╩ ╩╚═╝
  'POST /api/v1/webhooks/receive-usage-analytics': { action: 'webhooks/receive-usage-analytics', csrf: false },
  '/api/v1/webhooks/github': { action: 'webhooks/receive-from-github', csrf: false },
  'POST /api/v1/webhooks/receive-from-stripe': { action: 'webhooks/receive-from-stripe', csrf: false },
  'POST /api/v1/webhooks/receive-from-customer-fleet-instance': { action: 'webhooks/receive-from-customer-fleet-instance', csrf: false},

  //  ╔═╗╔═╗╦  ╔═╗╔╗╔╔╦╗╔═╗╔═╗╦╔╗╔╔╦╗╔═╗
  //  ╠═╣╠═╝║  ║╣ ║║║ ║║╠═╝║ ║║║║║ ║ ╚═╗
  //  ╩ ╩╩  ╩  ╚═╝╝╚╝═╩╝╩  ╚═╝╩╝╚╝ ╩ ╚═╝
  // Note that, in this app, these API endpoints may be accessed using the `Cloud.*()` methods
  // from the Parasails library, or by using those method names as the `action` in <ajax-form>.
  'POST /api/v1/deliver-contact-form-message':        { action: 'deliver-contact-form-message' },
  'POST /api/v1/entrance/send-password-recovery-email': { action: 'entrance/send-password-recovery-email' },
  'POST /api/v1/customers/signup':                     { action: 'entrance/signup' },
  'POST /api/v1/account/update-profile':               { action: 'account/update-profile' },
  'POST /api/v1/account/update-password':              { action: 'account/update-password' },
  'POST /api/v1/account/update-billing-card':          { action: 'account/update-billing-card'},
  'POST /api/v1/customers/login':                      { action: 'entrance/login' },
  '/api/v1/account/logout':                            { action: 'account/logout' },
  'POST /api/v1/customers/create-quote':               { action: 'customers/create-quote' },
  'POST /api/v1/customers/save-billing-info-and-subscribe': { action: 'customers/save-billing-info-and-subscribe' },
  'POST /api/v1/entrance/update-password-and-login':    { action: 'entrance/update-password-and-login' },
  'POST /api/v1/deliver-demo-signup':                   { action: 'deliver-demo-signup' },
  'POST /api/v1/create-or-update-one-newsletter-subscription': { action: 'create-or-update-one-newsletter-subscription' },
  '/api/v1/unsubscribe-from-all-newsletters': { action: 'unsubscribe-from-all-newsletters' },
  'POST /api/v1/admin/generate-license-key': { action: 'admin/generate-license-key' },
  'POST /api/v1/create-vanta-authorization-request': { action: 'create-vanta-authorization-request' },
  'POST /api/v1/deliver-mdm-beta-signup':                   { action: 'deliver-mdm-beta-signup' },
  'POST /api/v1/deliver-apple-csr ': { action: 'deliver-apple-csr', csrf: false},
  'POST /api/v1/deliver-premium-upgrade-form': { action: 'deliver-premium-upgrade-form' },
  'POST /api/v1/deliver-launch-party-signup':          { action: 'imagine/deliver-launch-party-signup' },
  'POST /api/v1/deliver-mdm-demo-email':               { action: 'deliver-mdm-demo-email' },
};

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
    locals: {
      isHomepage: true,
    }
  },

  'GET /contact': {
    action: 'view-contact',
    locals: {
      pageTitleForMeta: 'Contact us',
      pageDescriptionForMeta: 'Get in touch with our team.',
      hideFooterLinks: true,
    }
  },

  'GET /try-fleet': {
    action: 'view-fleetctl-preview',
    locals: {
      hideHeaderLinks: true,
      hideFooterLinks: true,
      pageTitleForMeta: 'fleetctl preview',
      pageDescriptionForMeta: 'Learn about getting started with Fleet using fleetctl.'
    }
  },

  'GET /pricing': {
    action: 'view-pricing',
    locals: {
      currentSection: 'pricing',
      pageTitleForMeta: 'Pricing',
      pageDescriptionForMeta: 'Use Fleet for free or get started with Fleet Premium (self-hosted or managed cloud). Have a large deployment? We\'ve got you covered.'
    }
  },

  'GET /logos': {
    action: 'view-press-kit',
    locals: {
      pageTitleForMeta: 'Logos',
      pageDescriptionForMeta: 'Download Fleet logos, wallpapers, and screenshots.'
    }
  },

  'GET /queries': {
    action: 'view-query-library',
    locals: {
      currentSection: 'documentation',
      pageTitleForMeta: 'Controls and policies',
      pageDescriptionForMeta: 'A growing collection of useful controls and policies for organizations deploying Fleet and osquery.'
    }
  },

  'GET /queries/:slug': {
    action: 'view-query-detail',// Meta title and description set in view action
    locals: {
      currentSection: 'documentation',
      // Note: this page's meta title and description are set in the page's view action
    }
  },

  'r|^/((success-stories|securing|releases|engineering|guides|announcements|podcasts|report|deploy)/(.+))$|': {
    skipAssets: false,
    action: 'articles/view-basic-article',// Meta title and description set in view action
  },// Handles /device-management/foo, /securing/foo, /releases/foo, /engineering/foo, /guides/foo, /announcements/foo, /deploy/foo, /podcasts/foo, /report/foo

  'r|^/((success-stories|securing|releases|engineering|guides|announcements|articles|podcasts|report|deploy))/*$|category': {
    skipAssets: false,
    action: 'articles/view-articles',// Meta title and description set in view action
  },// Handles the article landing page /articles, and the article cateogry pages (e.g. /device-management, /securing, /releases, etc)

  'GET /docs/?*': {
    skipAssets: false,
    action: 'docs/view-basic-documentation',// Meta title and description set in view action
    locals: {
      hideStartCTA: true,
      currentSection: 'documentation',
    }
  },// handles /docs and /docs/foo/bar

  'GET /handbook/?*':  {
    skipAssets: false,
    action: 'handbook/view-basic-handbook',// Meta title and description set in view action
    locals: {
      currentSection: 'community',
    }
  },// handles /handbook and /handbook/foo/bar

  'GET /new-license': {
    action: 'customers/view-new-license',
    locals: {
      hideHeaderLinks: true,
      hideFooterLinks: true,
      hideStartCTA: true,
      pageTitleForMeta: 'Get Fleet Premium',
      pageDescriptionForMeta: 'Generate your quote and start using Fleet Premium today.',
    }
  },
  'GET /register': {
    action: 'entrance/view-signup',
    locals: {
      hideFooterLinks: true,
      pageTitleForMeta: 'Sign up',
      pageDescriptionForMeta: 'Sign up for a Fleet account.',
    }
  },
  'GET /login': {
    action: 'entrance/view-login',
    locals: {
      hideFooterLinks: true,
      pageTitleForMeta: 'Log in',
      pageDescriptionForMeta: 'Log in to Fleet.',
    }
  },
  'GET /customers/dashboard': {
    action: 'customers/view-dashboard',
    locals: {
      hideHeaderLinks: true,
      hideFooterLinks: true,
      hideStartCTA: true,
      pageTitleForMeta: 'Customer dashboard',
      pageDescriptionForMeta: 'View and edit information about your Fleet Premium license.',
    }
  },
  'GET /customers/forgot-password': {
    action: 'entrance/view-forgot-password',
    locals: {
      hideHeaderLinks: true,
      hideFooterLinks: true,
      hideStartCTA: true,
      pageTitleForMeta: 'Forgot password',
      pageDescriptionForMeta: 'Recover the password for your Fleet customer account.',
    }
  },
  'GET /customers/new-password': {
    action: 'entrance/view-new-password',
    locals: {
      hideHeaderLinks: true,
      hideFooterLinks: true,
      hideStartCTA: true,
      pageTitleForMeta: 'New password',
      pageDescriptionForMeta: 'Change the password for your Fleet customer account.',
    }
  },

  'GET /reports/state-of-device-management': {
    action: 'reports/view-state-of-device-management',
    locals: {
      pageTitleForMeta: 'State of device management',
      pageDescriptionForMeta: 'We surveyed 200+ security practitioners to discover the state of device management in 2022. Click here to learn about their struggles and best practices.',
    }
  },

  'GET /admin/email-preview': {
    action: 'admin/view-email-templates',
    locals: {
      hideFooterLinks: true,
      showAdminLinks: true,
      hideStartCTA: true,
    },
  },

  'GET /admin/email-preview/*': {
    action: 'admin/view-email-template-preview',
    skipAssets: true,
    locals: {
      hideFooterLinks: true,
      showAdminLinks: true,
      hideStartCTA: true,
    },
  },

  'GET /admin/sandbox-waitlist': {
    action: 'admin/view-sandbox-waitlist',
    locals: {
      hideFooterLinks: true,
      showAdminLinks: true,
      hideStartCTA: true,
    },
  },

  'GET /tables/:tableName': {
    action: 'view-osquery-table-details',// Meta title and description set in view action
    locals: {
      currentSection: 'documentation',
    }
  },

  'GET /admin/generate-license': {
    action: 'admin/view-generate-license',
    locals: {
      hideFooterLinks: true,
      showAdminLinks: true,
      hideStartCTA: true,
    }
  },


  'GET /connect-vanta': {
    action: 'view-connect-vanta',
  },

  'GET /vanta-authorization': {
    action: 'view-vanta-authorization',
  },

  'GET /device-management': {
    action: 'view-device-management',
    locals: {
      pageTitleForMeta: 'Device management (MDM)',
      pageDescriptionForMeta: 'Manage your devices in any browser or use git to make changes as code.',
      currentSection: 'platform',
    }
  },

  'GET /observability': {
    action: 'view-observability',
    locals: {
      pageTitleForMeta: 'Observability',
      pageDescriptionForMeta: 'Pulse check anything, build reports, and ship data to any platform with Fleet.',
      currentSection: 'platform',
    }
  },

  'GET /software-management': {
    action: 'view-software-management',
    locals: {
      pageTitleForMeta: 'Software management',
      pageDescriptionForMeta: 'Pick from a curated app library or upload your own custom packages. Configure custom installation scripts if you need or let Fleet do it for you.',
      currentSection: 'platform',
    }
  },

  'GET /support': {
    action: 'view-support',
    locals: {
      pageTitleForMeta: 'Support',
      pageDescriptionForMeta: 'Ask a question, chat with engineers, or get in touch with the Fleet team.',
      currentSection: 'documentation',
    }
  },

  'GET /integrations': {
    action: 'view-integrations',
    locals: {
      pageTitleForMeta: 'Integrations',
      pageDescriptionForMeta: 'Integrate IT ticketing systems, SIEM and SOAR platforms, custom IT workflows, and more.',
      currentSection: 'platform'
    }
  },

  'GET /start': {
    action: 'view-start',
    locals: {
      hideFooterLinks: true,
      hideGetStartedButton: true,
      hideStartCTA: true,
      pageTitleForMeta: 'Start',
      pageDescriptionForMeta: 'Get Started with Fleet. Spin up a local demo or get your Premium license key.',
    }
  },

  'GET /better': {
    action: 'view-transparency',
    locals: {
      pageDescriptionForMeta: 'Discover how Fleet simplifies IT and security, prioritizing privacy, transparency, and trust for end users.',
      pageTitleForMeta: 'Better with Fleet'
    }
  },

  'GET /deals': {
    action: 'view-deals',
    locals: {
      pageTitleForMeta: 'Deal registration',
      pageDescriptionForMeta: 'Register an opportunity with a potential customer.',
      hideFooterLinks: true,
      hideStartCTA: true,
    }
  },

  'GET /customer-stories': {
    action: 'view-testimonials',
    locals: {
      pageTitleForMeta: 'Customer stories',
      pageDescriptionForMeta: 'See what people are saying about Fleet'
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
  'GET /try': '/get-started',
  'GET /docs/deploying/fleet-public-load-testing': '/docs/deploying/load-testing',
  'GET /handbook/customer-experience': '/handbook/customers',
  'GET /handbook/brand': '/handbook/digital-experience',
  'GET /guides/deploying-fleet-on-aws-with-terraform': '/deploy/deploying-fleet-on-aws-with-terraform',
  'GET /guides/deploying-fleet-on-render': '/deploy/deploying-fleet-on-render',
  'GET /use-cases/correlate-network-connections-with-community-id-in-osquery': '/guides/correlate-network-connections-with-community-id-in-osquery',
  'GET /use-cases/converting-unix-timestamps-with-osquery': '/guides/converting-unix-timestamps-with-osquery',
  'GET /use-cases/ebpf-the-future-of-osquery-on-linux': '/securing/ebpf-the-future-of-osquery-on-linux',
  'GET /use-cases/fleet-quick-tips-querying-procdump-eula-has-been-accepted': '/guides/fleet-quick-tips-querying-procdump-eula-has-been-accepted',
  'GET /use-cases/generate-process-trees-with-osquery': '/guides/generate-process-trees-with-osquery',
  'GET /use-cases/get-and-stay-compliant-across-your-devices-with-fleet': '/securing/get-and-stay-compliant-across-your-devices-with-fleet',
  'GET /use-cases/import-and-export-queries-and-packs-in-fleet': '/guides/import-and-export-queries-and-packs-in-fleet',
  'GET /guides/import-and-export-queries-and-packs-in-fleet': '/guides/import-and-export-queries-in-fleet',
  'GET /guides/deploy-security-agents': '/guides/deploy-software-packages',
  'GET /use-cases/locate-assets-with-osquery': '/guides/locate-assets-with-osquery',
  'GET /use-cases/osquery-a-tool-to-easily-ask-questions-about-operating-systems': '/guides/osquery-a-tool-to-easily-ask-questions-about-operating-systems',
  'GET /use-cases/osquery-consider-joining-against-the-users-table': '/guides/osquery-consider-joining-against-the-users-table',
  'GET /use-cases/stay-on-course-with-your-security-compliance-goals': '/guides/stay-on-course-with-your-security-compliance-goals',
  'GET /use-cases/using-elasticsearch-and-kibana-to-visualize-osquery-performance': '/guides/using-elasticsearch-and-kibana-to-visualize-osquery-performance',
  'GET /use-cases/work-may-be-watching-but-it-might-not-be-as-bad-as-you-think': '/securing/work-may-be-watching-but-it-might-not-be-as-bad-as-you-think',
  'GET /docs/contributing/testing':  '/docs/contributing/testing-and-local-development',
  'GET /handbook/people/ceo-handbook': '/handbook/ceo',
  'GET /handbook/company/ceo-handbook': '/handbook/ceo',
  'GET /handbook/growth': '/handbook/marketing#growth',
  'GET /handbook/community': '/handbook/marketing#community',
  'GET /handbook/digital-experience/article-formatting-guide': '/handbook/marketing/article-formatting-guide',
  'GET /handbook/marketing/commonly-used-terms': '/handbook/company/communications#commonly-used-terms',
  'GET /handbook/marketing/markdown-guide': '/handbook/company/communications#writing-in-fleet-flavored-markdown',
  'GET /handbook/digital-experience/commonly-used-terms': '/handbook/company/communications#commonly-used-terms',
  'GET /handbook/digital-experience/how-to-submit-and-publish-an-article': '/handbook/marketing/how-to-submit-and-publish-an-article',
  'GET /handbook/digital-experience/markdown-guide': '/handbook/company/communications#writing-in-fleet-flavored-markdown',
  'GET /handbook/ceo': '/handbook/digital-experience',
  'GET /handbook/marketing/content-style-guide': '/handbook/company/communications#writing',
  'GET /handbook/marketing/editor-guide/': '/handbook/company/communications#github',
  'GET /handbook/marketing/docs-handbook/': '/handbook/company/communications#docs',
  'GET /handbook/marketing/website-handbook/': '/handbook/company/communications#website',
  'GET /handbook/quality': '/handbook/engineering#quality',
  'GET /device-management/fleet-user-stories-f100': '/success-stories/fleet-user-stories-wayfair',
  'GET /device-management/fleet-user-stories-schrodinger': '/success-stories/fleet-user-stories-wayfair',
  'GET /device-management/fleet-user-stories-wayfair': '/success-stories/fleet-user-stories-wayfair',
  'GET /handbook/security': '/handbook/digital-experience/security',
  'GET /handbook/security/security-policies':'/handbook/digital-experience/security',// « reasoning: https://github.com/fleetdm/fleet/pull/9624
  'GET /handbook/business-operations/security-policies':'/handbook/digital-experience/security',
  'GET /handbook/business-operations/application-security': '/handbook/digital-experience/security',
  'GET /handbook/business-operations/security-audits': '/handbook/digital-experience/security',
  'GET /handbook/business-operations/security': '/handbook/digital-experience/security',
  'GET /handbook/business-operations/vendor-questionnaires': '/handbook/digital-experience/security',
  'GET /handbook/handbook': '/handbook/company/handbook',
  'GET /handbook/company/development-groups': '/handbook/company/product-groups',
  'GET /docs/using-fleet/mdm-macos-settings': '/docs/using-fleet/mdm-custom-macos-settings',
  'GET /platform': '/',
  'GET /handbook/company/senior-software-backend-engineer': 'https://www.linkedin.com/posts/mikermcneil_in-addition-to-our-product-quality-specialist-activity-7067711903166279680-6CMH',
  'GET /handbook/business-operations/ceo-handbook': '/handbook/ceo',
  'GET /handbook/business-operations/people-operations': '/handbook/company/communications#hiring',
  'GET /handbook/marketing': '/handbook/demand/',
  'GET /handbook/customers': '/handbook/sales/',
  'GET /handbook/product': '/handbook/product-design',
  'GET /handbook/business-operations': '/handbook/finance',

  'GET /docs': '/docs/get-started/why-fleet',
  'GET /docs/get-started': '/docs/get-started/why-fleet',
  'GET /docs/rest-api': '/docs/rest-api/rest-api',
  'GET /docs/configuration': '/docs/configuration/fleet-server-configuration',
  'GET /docs/contributing': 'https://github.com/fleetdm/fleet/tree/main/docs/Contributing',
  'GET /docs/deploy': '/docs/deploy/introduction',
  'GET /docs/using-fleet/faq': '/docs/get-started/faq',
  'GET /docs/using-fleet/monitoring-fleet': '/docs/deploy/monitoring-fleet',
  'GET /docs/using-fleet/adding-hosts': '/docs/using-fleet/enroll-hosts',
  'GET /docs/using-fleet/fleetd': '/docs/using-fleet/enroll-hosts',
  'GET /docs/using-fleet/teams': '/docs/using-fleet/segment-hosts',
  'GET /docs/using-fleet/permissions': '/docs/using-fleet/manage-access',
  'GET /docs/using-fleet/chromeos': '/docs/using-fleet/enroll-chromebooks',
  'GET /docs/using-fleet/rest-api': '/docs/rest-api/rest-api',
  'GET /docs/using-fleet/configuration-files': '/docs/configuration/configuration-files/',
  'GET /docs/using-fleet/application-security': '/handbook/digital-experience/application-security',
  'GET /docs/using-fleet/security-audits': '/handbook/digital-experience/security-audits',
  'GET /docs/using-fleet/process-file-events': '/guides/querying-process-file-events-table-on-centos-7',
  'GET /docs/using-fleet/audit-activities': '/docs/using-fleet/audit-logs',
  'GET /docs/using-fleet/detail-queries-summary': '/docs/using-fleet/understanding-host-vitals',
  'GET /docs/using-fleet/orbit': '/docs/using-fleet/enroll-hosts',
  'GET /docs/deploying': '/docs/deploy',
  'GET /docs/deploying/faq': '/docs/get-started/faq',
  'GET /docs/deploying/introduction': '/docs/deploy/introduction',
  'GET /docs/deploy/introduction': '/docs/deploy/deploy-fleet',
  'GET /docs/deploying/reference-architectures': '/docs/deploy/reference-architectures',
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
  'GET /deploy/deploying-fleet-on-render': '/docs/deploy/deploy-on-render',
  'GET /deploy/deploy-fleet-on-hetzner-cloud': '/docs/deploy/deploy-on-hetzner-cloud',
  'GET /deploy': '/docs/deploy',
  'GET /deploy/deploying-fleet-on-aws-with-terraform': '/docs/deploy/deploy-on-aws-with-terraform',
  'GET /docs/deploy/server-installation': '/docs/deploy/introduction',
  'GET /handbook/company/ceo': '/handbook/ceo',
  'GET /handbook/communications': '/handbook/company/communications',
  'GET /handbook/leadership': '/handbook/company/leadership',
  'GET /handbook/product-groups': '/handbook/company/product-groups',
  'GET /handbook/company/customer-solutions-architect': '/handbook/company/open-positions/customer-solutions-architect',
  'GET /handbook/company/software-engineer': '/handbook/company/open-positions/software-engineer',
  'GET /handbook/company/software-engineer-windows-go': '/handbook/company/open-positions/software-engineer-windows-go',
  'GET /osquery-management': '/endpoint-ops',
  'GET /guides/using-github-actions-to-apply-configuration-profiles-with-fleet': 'https://github.com/fleetdm/fleet-gitops',
  'GET /docs/using-fleet/mdm-macos-updates': '/docs/using-fleet/mdm-os-updates',
  'GET /example-windows-profile': 'https://github.com/fleetdm/fleet-gitops/blob/860dcf2609e2b25a6d6becf8006a7118a19cd615/lib/windows-screenlock.xml',// « resuable link for OS settings doc page
  'GET /docs/using-fleet/mdm-custom-macos-settings': '/docs/using-fleet/mdm-custom-os-settings',
  'GET /customers/login': '/login',
  'GET /customers/register': '/register',
  'GET /try-fleet/login': '/login',
  'GET /try-fleet/register': '/register',
  'GET /customers/new-license': '/new-license',
  'GET /try-fleet/fleetctl-preview': '/try-fleet',
  'GET /upgrade': '/pricing',
  'GET /docs/deploy/system-d': '/docs/deploy/reference-architectures#systemd',
  'GET /docs/deploy/proxies': '/docs/deploy/reference-architectures#using-a-proxy',
  'GET /docs/deploy/public-ip': '/docs/deploy/reference-architectures#public-ips-of-devices',
  'GET /docs/deploy/monitoring-fleet': '/docs/deploy/reference-architectures#monitoring-fleet',
  'GET /docs/deploy/deploy-fleet-on-aws-ecs': '/guides/deploy-fleet-on-aws-ecs',
  'GET /docs/deploy/deploy-fleet-on-centos': '/guides/deploy-fleet-on-centos',
  'GET /docs/deploy/cloudgov': '/guides/deploy-fleet-on-cloudgov',
  'GET /docs/deploy/deploy-on-aws-with-terraform': '/guides/deploy-fleet-on-aws-with-terraform',
  'GET /docs/deploy/deploy-on-hetzner-cloud': '/guides/deploy-fleet-on-hetzner-cloud',
  'GET /docs/deploy/deploy-on-render': '/guides/deploy-fleet-on-render',
  'GET /docs/deploy/deploy-fleet-on-kubernetes': '/guides/deploy-fleet-on-kubernetes',
  'GET /docs/using-fleet/mdm-macos-setup': '/docs/using-fleet/mdm-setup',
  'GET /transparency': '/better',
  'GET /docs/configuration/configuration-files': '/docs/using-fleet/gitops',
  'GET /try-fleet/explore-data': '/tables/account_policy_data',
  'GET /try-fleet/explore-data/:hostPlatform/:tableName': {
    fn: (req, res)=>{
      return res.redirect('/tables/'+req.param('tableName'));
    }
  },
  'GET /docs/using-fleet/fleet-ui': (req,res)=> { return res.redirect(301, '/guides/queries');},
  'GET /docs/using-fleet/learn-how-to-use-fleet': (req,res)=> { return res.redirect(301, '/guides/queries');},
  'GET /docs/using-fleet/fleetctl-cli': (req,res)=> { return res.redirect(301, '/guides/fleetctl');},
  'GET /docs/using-fleet/fleet-desktop': (req,res)=> { return res.redirect(301, '/guides/fleet-desktop');},
  'GET /docs/using-fleet/enroll-hosts': (req,res)=> { return res.redirect(301, '/guides/enroll-hosts');},
  'GET /docs/using-fleet/manage-access': (req,res)=> { return res.redirect(301, '/guides/role-based-access');},
  'GET /docs/using-fleet/segment-hosts': (req,res)=> { return res.redirect(301, '/guides/teams');},
  'GET /docs/using-fleet/supported-browsers': (req,res)=> { return res.redirect(301, '/docs/get-started/faq');},
  'GET /docs/using-fleet/supported-host-operating-systems': (req,res)=> { return res.redirect(301, '/docs/get-started/faq');},
  'GET /docs/using-fleet/gitops': (req,res)=> { return res.redirect(301, '/docs/configuration/yaml-files');},
  'GET /docs/using-fleet/mdm-setup': (req,res)=> { return res.redirect(301, '/guides/macos-mdm-setup');},
  'GET /docs/using-fleet/mdm-migration-guide': (req,res)=> { return res.redirect(301, '/guides/mdm-migration');},
  'GET /docs/using-fleet/mdm-os-updates': (req,res)=> { return res.redirect(301, '/guides/enforce-os-updates');},
  'GET /docs/using-fleet/mdm-disk-encryption': (req,res)=> { return res.redirect(301, '/guides/enforce-disk-encryption');},
  'GET /docs/using-fleet/mdm-custom-os-settings': (req,res)=> { return res.redirect(301, '/guides/custom-os-settings');},
  'GET /docs/using-fleet/mdm-macos-setup-experience': (req,res)=> { return res.redirect(301, '/guides/macos-setup-experience');},
  'GET /docs/using-fleet/scripts': (req,res)=> { return res.redirect(301, '/guides/scripts');},
  'GET /docs/using-fleet/automations': (req,res)=> { return res.redirect(301, '/guides/automations');},
  'GET /docs/using-fleet/puppet-module': (req,res)=> { return res.redirect(301, '/guides/puppet-module');},
  'GET /docs/using-fleet/vulnerability-processing': (req,res)=> { return res.redirect(301, '/guides/vulnerability-processing');},
  'GET /docs/using-fleet/cis-benchmarks': (req,res)=> { return res.redirect(301, '/guides/cis-benchmarks');},
  'GET /docs/using-fleet/osquery-process': (req,res)=> { return res.redirect(301, '/guides/osquery-watchdog');},
  'GET /docs/using-fleet/update-agents': (req,res)=> { return res.redirect(301, '/guides/fleetd-updates');},
  'GET /docs/using-fleet/usage-statistics': (req,res)=> { return res.redirect(301, '/guides/fleet-usage-statistics');},
  'GET /docs/using-fleet/downgrading-fleet': (req,res)=> { return res.redirect(301, '/guides/downgrade-fleet');},
  'GET /docs/using-fleet/enroll-chromebooks': (req,res)=> { return res.redirect(301, '/guides/chrome-os');},
  'GET /docs/using-fleet/audit-logs': (req,res)=> { return res.redirect(301, 'https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Audit-logs.md');},
  'GET /docs/using-fleet/understanding-host-vitals': (req,res)=> { return res.redirect(301, 'https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Understanding-host-vitals.md');},
  'GET /docs/using-fleet/standard-query-library': (req,res)=> { return res.redirect(301, '/guides/standard-query-library');},
  'GET /docs/using-fleet/mdm-commands': (req,res)=> { return res.redirect(301, '/guides/mdm-commands');},
  'GET /docs/using-fleet/log-destinations': (req,res)=> { return res.redirect(301, '/guides/log-destinations');},
  'GET /guides/how-to-uninstall-osquery': (req,res)=> { return res.redirect(301, '/guides/how-to-uninstall-fleetd');},
  'GET /guides/sysadmin-diaries-lost-device': (req,res)=> { return res.redirect(301, '/guides/lock-wipe-hosts');},

  //  ╔╦╗╦╔═╗╔═╗  ╦═╗╔═╗╔╦╗╦╦═╗╔═╗╔═╗╔╦╗╔═╗   ┬   ╔╦╗╔═╗╦ ╦╔╗╔╦  ╔═╗╔═╗╔╦╗╔═╗
  //  ║║║║╚═╗║    ╠╦╝║╣  ║║║╠╦╝║╣ ║   ║ ╚═╗  ┌┼─   ║║║ ║║║║║║║║  ║ ║╠═╣ ║║╚═╗
  //  ╩ ╩╩╚═╝╚═╝  ╩╚═╚═╝═╩╝╩╩╚═╚═╝╚═╝ ╩ ╚═╝  └┘   ═╩╝╚═╝╚╩╝╝╚╝╩═╝╚═╝╩ ╩═╩╝╚═╝

  // Convenience
  // =============================================================================================================
  // Things that people are used to typing in to the URL and just randomly trying.
  //
  // For example, a clever user might try to visit fleetdm.com/documentation, not knowing that Fleet's website
  // puts this kind of thing under /docs, NOT /documentation.  These "convenience" redirects are to help them out.
  'GET /testimonials':               '/customer-stories',
  'GET /admin':                      '/admin/email-preview',
  'GET /renew':                      'https://calendly.com/zayhanlon/fleet-renewal-discussion',
  'GET /documentation':              '/docs',
  'GET /contribute':                 '/docs/contributing',
  'GET /install':                    '/fleetctl-preview',
  'GET /company':                    '/company/about',
  'GET /company/about':              '/handbook', // FUTURE: brief "about" page explaining the origins of the company
  'GET /company/contact':            '/contact',
  'GET /legal':                      '/legal/terms',
  'GET /terms':                      '/legal/terms',
  'GET /handbook/security/github':   '/handbook/security#git-hub-security',
  'GET /slack':                      '/support',// Note: This redirect is used on error pages and email templates in the Fleet UI.
  'GET /docs/using-fleet/updating-fleet': '/docs/deploying/upgrading-fleet',
  'GET /blog':                   '/articles',
  'GET /brand':                  '/logos',
  'GET /get-started':            '/try-fleet',
  'GET /g':                       (req,res)=> { let originalQueryStringWithAmp = req.url.match(/\?(.+)$/) ? '&'+req.url.match(/\?(.+)$/)[1] : ''; return res.redirect(301, sails.config.custom.baseUrl+'/?meet-fleet'+originalQueryStringWithAmp); },
  'GET /test-fleet-sandbox':     '/register',
  'GET /unsubscribe':             (req,res)=> { let originalQueryString = req.url.match(/\?(.+)$/) ? req.url.match(/\?(.+)$/)[1] : ''; return res.redirect(301, sails.config.custom.baseUrl+'/api/v1/unsubscribe-from-marketing-emails?'+originalQueryString);},
  'GET /unsubscribe-from-newsletter':             (req,res)=> { let originalQueryString = req.url.match(/\?(.+)$/) ? req.url.match(/\?(.+)$/)[1] : ''; return res.redirect(301, sails.config.custom.baseUrl+'/api/v1/unsubscribe-from-all-newsletters?'+originalQueryString);},
  'GET /tables':                 '/tables/account_policy_data',
  'GET /imagine/launch-party':  'https://www.eventbrite.com/e/601763519887',
  'GET /blackhat2023':   'https://github.com/fleetdm/fleet/tree/main/tools/blackhat-mdm', // Assets from @marcosd4h & @zwass Black Hat 2023 talk
  'GET /fleetctl-preview':   '/try-fleet',
  'GET /try-fleet/sandbox-expired':   '/try-fleet',
  'GET /try-fleet/sandbox':   '/try-fleet',
  'GET /try-fleet/waitlist':   '/try-fleet',
  'GET /endpoint-operations': '/endpoint-ops',// « just in case we type it the wrong way
  'GET /example-dep-profile': 'https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/automatic-enrollment.dep.json',
  'GET /vulnerability-management': (req,res)=> { let originalQueryString = req.url.match(/\?(.+)$/) ? '?'+req.url.match(/\?(.+)$/)[1] : ''; return res.redirect(301, sails.config.custom.baseUrl+'/software-management'+originalQueryString);},
  'GET /endpoint-ops': (req,res)=> { let originalQueryString = req.url.match(/\?(.+)$/) ? '?'+req.url.match(/\?(.+)$/)[1] : ''; return res.redirect(301, sails.config.custom.baseUrl+'/observability'+originalQueryString);},

  // Shortlinks for texting friends, radio ads, etc
  'GET /mdm': '/device-management?utm_content=mdm',// « alias for radio ad
  'GET /it': '/observability?utm_content=eo-it',
  'GET /seceng': '/observability?utm_content=eo-security',
  'GET /vm': '/software-management?utm_content=vm',

  // Fleet UI
  // =============================================================================================================
  // Redirects for external links from the Fleet UI & CLI, including to fleetdm.com and to external websites not
  // maintained by Fleet. These help avoid broken links by reducing surface area of links to maintain in the UI.
  'GET /learn-more-about/chromeos-updates': 'https://support.google.com/chrome/a/answer/6220366',
  'GET /learn-more-about/just-in-time-provisioning': '/docs/deploy/single-sign-on-sso#just-in-time-jit-user-provisioning',
  'GET /learn-more-about/os-updates': '/docs/using-fleet/mdm-os-updates',
  'GET /sign-in-to/microsoft-automatic-enrollment-tool': 'https://portal.azure.com',
  'GET /learn-more-about/custom-os-settings': '/docs/using-fleet/mdm-custom-os-settings',
  'GET /learn-more-about/ndes': 'https://learn.microsoft.com/en-us/windows-server/identity/ad-cs/network-device-enrollment-service-overview', // TODO: Confirm URL
  'GET /learn-more-about/idp-email': 'https://fleetdm.com/docs/rest-api/rest-api#get-human-device-mapping',
  'GET /learn-more-about/enrolling-hosts': '/docs/using-fleet/adding-hosts',
  'GET /learn-more-about/setup-assistant': '/docs/using-fleet/mdm-macos-setup-experience#macos-setup-assistant',
  'GET /learn-more-about/policy-automations': '/docs/using-fleet/automations',
  'GET /install-wine': 'https://github.com/fleetdm/fleet/blob/main/scripts/macos-install-wine.sh',
  'GET /learn-more-about/creating-service-accounts': 'https://console.cloud.google.com/projectselector2/iam-admin/serviceaccounts/create?walkthrough_id=iam--create-service-account&pli=1#step_index=1',
  'GET /learn-more-about/google-workspace-domains': 'https://admin.google.com/ac/domains/manage',
  'GET /learn-more-about/domain-wide-delegation': 'https://admin.google.com/ac/owl/domainwidedelegation',
  'GET /learn-more-about/enabling-calendar-api': 'https://console.cloud.google.com/apis/library/calendar-json.googleapis.com',
  'GET /learn-more-about/downgrading': '/docs/using-fleet/downgrading-fleet',
  'GET /learn-more-about/fleetd': '/docs/get-started/anatomy#fleetd',
  'GET /learn-more-about/rotating-enroll-secrets': 'https://github.com/fleetdm/fleet/blob/main/docs/Contributing/fleetctl-apply.md#rotating-enroll-secrets',
  'GET /learn-more-about/audit-logs': '/docs/using-fleet/audit-logs',
  'GET /learn-more-about/calendar-events': '/announcements/fleet-in-your-calendar-introducing-maintenance-windows',
  'GET /learn-more-about/setup-windows-mdm': '/guides/windows-mdm-setup',
  'GET /learn-more-about/setup-abm': '/docs/using-fleet/mdm-setup#apple-business-manager-abm',
  'GET /learn-more-about/renew-apns': '/docs/using-fleet/mdm-setup#apple-push-notification-service-apns',
  'GET /learn-more-about/renew-abm': '/docs/using-fleet/mdm-setup#apple-business-manager-abm',
  'GET /learn-more-about/fleet-server-private-key': '/docs/configuration/fleet-server-configuration#server-private-key',
  'GET /learn-more-about/agent-options': '/docs/configuration/agent-configuration',
  'GET /learn-more-about/enable-user-collection': '/docs/using-fleet/gitops#features',
  'GET /learn-more-about/host-identifiers': '/docs/rest-api/rest-api#get-host-by-identifier',
  'GET /learn-more-about/uninstall-fleetd': '/docs/using-fleet/faq#how-can-i-uninstall-fleetd',
  'GET /learn-more-about/vulnerability-processing': '/docs/using-fleet/vulnerability-processing',
  'GET /learn-more-about/dep-profile': 'https://developer.apple.com/documentation/devicemanagement/define_a_profile',
  'GET /learn-more-about/apple-business-manager-tokens-api': '/docs/rest-api/rest-api#list-apple-business-manager-abm-tokens',
  'GET /learn-more-about/apple-business-manager-teams-api': 'https://github.com/fleetdm/fleet/blob/main/docs/Contributing/API-for-contributors.md#update-abm-tokens-teams',
  'GET /learn-more-about/apple-business-manager-gitops': '/docs/using-fleet/gitops#apple-business-manager',
  'GET /learn-more-about/s3-bootstrap-package': '/docs/configuration/fleet-server-configuration#s-3-software-installers-bucket',
  'GET /learn-more-about/available-os-update-versions': '/guides/enforce-os-updates#available-macos-ios-and-ipados-versions',
  'GET /learn-more-about/policy-automation-install-software': '/guides/automatic-software-install-in-fleet',
  'GET /learn-more-about/exe-install-scripts': '/guides/exe-install-scripts',
  'GET /learn-more-about/install-scripts': '/guides/deploy-software-packages#install-script',
  'GET /learn-more-about/uninstall-scripts': '/guides/deploy-software-packages#uninstall-script',
  'GET /learn-more-about/read-package-version': '/guides/deploy-software-packages#add-a-software-package-to-a-team',
  'GET /learn-more-about/fleetctl': '/guides/fleetctl',
  'GET /feature-request': 'https://github.com/fleetdm/fleet/issues/new?assignees=&labels=~feature+fest%2C%3Aproduct&projects=&template=feature-request.md&title=',
  'GET /learn-more-about/policy-automation-run-script': '/guides/policy-automation-run-script',
  'GET /learn-more-about/installing-fleetctl': '/guides/fleetctl#installing-fleetctl',
  'GET /contribute-to/policies': 'https://github.com/fleetdm/fleet/edit/main/docs/01-Using-Fleet/standard-query-library/standard-query-library.yml',

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
  'GET /community':              'https://join.slack.com/t/osquery/shared_invite/zt-2op37v6qp-aVPivU5xB_FwuYElN0Z1lw',

  // Temporary redirects
  // =============================================================================================================
  // For events, etc. that can be removed after a certain date. Please leave a comment with a valid until date.
  'GET /rsaparty':               'https://www.eventbrite.com/e/fleet-launch-party-at-rsac-tickets-877549332677?aff=fleetdm', // Valid until 2024-05-09
  'GET /rsavip':                 'https://www.eventbrite.com/e/fleet-launch-party-at-rsac-tickets-877549332677?aff=fleetdm&discount=Fleet2024', // Valid until 2024-05-09


  //  ╦ ╦╔═╗╔╗ ╦ ╦╔═╗╔═╗╦╔═╔═╗
  //  ║║║║╣ ╠╩╗╠═╣║ ║║ ║╠╩╗╚═╗
  //  ╚╩╝╚═╝╚═╝╩ ╩╚═╝╚═╝╩ ╩╚═╝
  'POST /api/v1/webhooks/receive-usage-analytics': { action: 'webhooks/receive-usage-analytics', csrf: false },
  '/api/v1/webhooks/github': { action: 'webhooks/receive-from-github', csrf: false },
  'POST /api/v1/webhooks/receive-from-stripe': { action: 'webhooks/receive-from-stripe', csrf: false },
  'POST /api/v1/get-est-device-certificate': { action: 'get-est-device-certificate', csrf: false},

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
  'POST /api/v1/admin/build-license-key': { action: 'admin/build-license-key' },
  'POST /api/v1/create-vanta-authorization-request': { action: 'create-vanta-authorization-request'},
  'POST /api/v1/create-external-vanta-authorization-request': { action: 'create-vanta-authorization-request', csrf: false },
  'GET /redirect-vanta-authorization-request': { action: 'redirect-vanta-authorization-request' },
  'POST /api/v1/deliver-mdm-beta-signup':                   { action: 'deliver-mdm-beta-signup' },
  'POST /api/v1/get-human-interpretation-from-osquery-sql': { action: 'get-human-interpretation-from-osquery-sql', csrf: false },
  'POST /api/v1/deliver-apple-csr ': { action: 'deliver-apple-csr', csrf: false},
  'POST /api/v1/deliver-mdm-demo-email':               { action: 'deliver-mdm-demo-email' },
  'POST /api/v1/admin/provision-sandbox-instance-and-deliver-email': { action: 'admin/provision-sandbox-instance-and-deliver-email' },
  'POST /api/v1/deliver-talk-to-us-form-submission': { action: 'deliver-talk-to-us-form-submission' },
  'POST /api/v1/save-questionnaire-progress': { action: 'save-questionnaire-progress' },
  'POST /api/v1/account/update-start-cta-visibility': { action: 'account/update-start-cta-visibility' },
  'POST /api/v1/deliver-deal-registration-submission': { action: 'deliver-deal-registration-submission' },
  '/api/v1/unsubscribe-from-marketing-emails': { action: 'unsubscribe-from-marketing-emails' },
};

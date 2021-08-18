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
  'GET /':                   { action: 'view-homepage-or-redirect', locals: { isHomepage: true } },
  'GET /company/contact':    { action:   'view-contact' },
  'GET /get-started':        { action:   'view-pricing' },

  'GET /install':            'https://github.com/fleetdm/fleet/blob/main/README.md', // « FUTURE: When ready, bring back { action:   'view-get-started' }
  '/hall-of-fame':           'https://github.com/fleetdm/fleet/pulse',
  '/company/about':          '/handbook', // FUTURE: brief "about" page explaining the origins of the company

  'GET /queries':            { action: 'view-query-library' },
  'GET /queries/:slug':      { action: 'view-query-detail' },

  'GET /docs/?*':            { skipAssets: false, action: 'docs/view-basic-documentation' },// handles /docs and /docs/foo/bar
  // 'GET /handbook/?*':        { skipAssets: false, action: 'handbook/view-basic-handbook' },// handles /handbook and /handbook/foo/bar
  'GET /handbook':           'https://github.com/fleetdm/fleet/tree/main/handbook',// TODO: Bring back the above when styles are ready

  '/contribute':             '/docs/contribute',
  '/company/stewardship':    'https://github.com/fleetdm/fleet', // FUTURE: page about how we approach open source and our commitments to the community
  '/legal/terms':            'https://docs.google.com/document/d/1OM6YDVIs7bP8wg6iA3VG13X086r64tWDqBSRudG4a0Y/edit',
  '/security':               'https://github.com/fleetdm/fleet/security/policy',

  'GET /transparency':       { action: 'view-transparency' },

  'GET /apply':              'https://fleet-device-management.breezy.hr',


  //  ╦  ╔═╗╔═╗╔═╗╔═╗╦ ╦  ╦═╗╔═╗╔╦╗╦╦═╗╔═╗╔═╗╔╦╗╔═╗
  //  ║  ║╣ ║ ╦╠═╣║  ╚╦╝  ╠╦╝║╣  ║║║╠╦╝║╣ ║   ║ ╚═╗
  //  ╩═╝╚═╝╚═╝╩ ╩╚═╝ ╩   ╩╚═╚═╝═╩╝╩╩╚═╚═╝╚═╝ ╩ ╚═╝
  //  ┌─  ┌─┐┌─┐┬─┐  ┌┐ ┌─┐┌─┐┬┌─┬ ┬┌─┐┬─┐┌┬┐┌─┐  ┌─┐┌─┐┌┬┐┌─┐┌─┐┌┬┐  ─┐
  //  │   ├┤ │ │├┬┘  ├┴┐├─┤│  ├┴┐│││├─┤├┬┘ ││└─┐  │  │ ││││├─┘├─┤ │    │
  //  └─  └  └─┘┴└─  └─┘┴ ┴└─┘┴ ┴└┴┘┴ ┴┴└──┴┘└─┘  └─┘└─┘┴ ┴┴  ┴ ┴ ┴o  ─┘
  // Use these redirects for deprecated/legacy links, so that they go to an appropriate new place instead of just being broken when docs/etc move or get renamed.
  // > Note that these redirects take precedence over less specific wildcard routes like '/docs/*' and '/handbook/*'

  'GET /docs/using-fleet/some-deprecated-link-like-this': '/docs/using-fleet/supported-browsers',// « this is just an example to show how

  //  ╔╦╗╦╔═╗╔═╗  ╦═╗╔═╗╔╦╗╦╦═╗╔═╗╔═╗╔╦╗╔═╗   ┬   ╔╦╗╔═╗╦ ╦╔╗╔╦  ╔═╗╔═╗╔╦╗╔═╗
  //  ║║║║╚═╗║    ╠╦╝║╣  ║║║╠╦╝║╣ ║   ║ ╚═╗  ┌┼─   ║║║ ║║║║║║║║  ║ ║╠═╣ ║║╚═╗
  //  ╩ ╩╩╚═╝╚═╝  ╩╚═╚═╝═╩╝╩╩╚═╚═╝╚═╝ ╩ ╚═╝  └┘   ═╩╝╚═╝╚╩╝╝╚╝╩═╝╚═╝╩ ╩═╩╝╚═╝

  // Convenience
  '/pricing':                    '/get-started',
  '/documentation':              '/docs',
  '/company':                    '/company/about',
  '/support':                    '/company/contact',
  '/contact':                    '/company/contact',
  '/legal':                      '/legal/terms',
  '/terms':                      '/legal/terms',

  // Sitemap
  'GET /sitemap.xml':            { action: 'download-sitemap' },

  // Blog
  '/blog':                       'https://blog.fleetdm.com',

  // Legacy (to avoid breaking links)
  '/try-fleet':                  '/get-started',


  //  ╦ ╦╔═╗╔╗ ╦ ╦╔═╗╔═╗╦╔═╔═╗
  //  ║║║║╣ ╠╩╗╠═╣║ ║║ ║╠╩╗╚═╗
  //  ╚╩╝╚═╝╚═╝╩ ╩╚═╝╚═╝╩ ╩╚═╝
  'POST /api/v1/webhooks/receive-usage-analytics': { action: 'webhooks/receive-usage-analytics', csrf: false },
  '/api/v1/webhooks/github': { action: 'webhooks/receive-from-github', csrf: false },


  //  ╔═╗╔═╗╦  ╔═╗╔╗╔╔╦╗╔═╗╔═╗╦╔╗╔╔╦╗╔═╗
  //  ╠═╣╠═╝║  ║╣ ║║║ ║║╠═╝║ ║║║║║ ║ ╚═╗
  //  ╩ ╩╩  ╩  ╚═╝╝╚╝═╩╝╩  ╚═╝╩╝╚╝ ╩ ╚═╝
  // Note that, in this app, these API endpoints may be accessed using the `Cloud.*()` methods
  // from the Parasails library, or by using those method names as the `action` in <ajax-form>.
  'POST  /api/v1/deliver-contact-form-message':          { action: 'deliver-contact-form-message' },

};

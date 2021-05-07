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

  '/blog':           'https://medium.com/fleetdm',

  '/legal/terms': 'https://docs.google.com/document/d/1OM6YDVIs7bP8wg6iA3VG13X086r64tWDqBSRudG4a0Y/edit',

  '/security': 'https://github.com/fleetdm/fleet/security/policy',

  '/company/about':          '/blog', // FUTURE: brief "about" page explaining the origins of the company
  '/company/stewardship':    'https://github.com/fleetdm/fleet', // FUTURE: page about how we approach open source and our commitments to the community
  'GET /company/contact':    { action:   'view-contact' },
  'GET /apply': 'https://fleet-device-management.breezy.hr',
  'GET /get-started': 'https://github.com/fleetdm/fleet/blob/master/README.md', // « FUTURE: When ready, bring back { action:   'view-get-started' }
  'GET /pricing':    { action:   'view-pricing' },
  '/documentation': 'https://github.com/fleetdm/fleet/tree/master/docs',
  '/contribute': 'https://github.com/fleetdm/fleet/tree/master/docs/4-Contribution',
  '/hall-of-fame': 'https://github.com/fleetdm/fleet/pulse',


  // 'GET /welcome/:unused?':   { action: 'dashboard/view-welcome' },

  // 'GET /faq':                { action:   'view-faq' },
  // 'GET /legal/terms':        { action:   'legal/view-terms' },
  // 'GET /legal/privacy':      { action:   'legal/view-privacy' },

  // 'GET /signup':             { action: 'entrance/view-signup' },
  // 'GET /email/confirm':      { action: 'entrance/confirm-email' },
  // 'GET /email/confirmed':    { action: 'entrance/view-confirmed-email' },

  // 'GET /login':              { action: 'entrance/view-login' },
  // 'GET /password/forgot':    { action: 'entrance/view-forgot-password' },
  // 'GET /password/new':       { action: 'entrance/view-new-password' },

  // 'GET /account':            { action: 'account/view-account-overview' },
  // 'GET /account/password':   { action: 'account/view-edit-password' },
  // 'GET /account/profile':    { action: 'account/view-edit-profile' },


  //  ╔╦╗╦╔═╗╔═╗  ╦═╗╔═╗╔╦╗╦╦═╗╔═╗╔═╗╔╦╗╔═╗   ┬   ╔╦╗╔═╗╦ ╦╔╗╔╦  ╔═╗╔═╗╔╦╗╔═╗
  //  ║║║║╚═╗║    ╠╦╝║╣  ║║║╠╦╝║╣ ║   ║ ╚═╗  ┌┼─   ║║║ ║║║║║║║║  ║ ║╠═╣ ║║╚═╗
  //  ╩ ╩╩╚═╝╚═╝  ╩╚═╚═╝═╩╝╩╩╚═╚═╝╚═╝ ╩ ╚═╝  └┘   ═╩╝╚═╝╚╩╝╝╚╝╩═╝╚═╝╩ ╩═╩╝╚═╝

  // Convenience
  '/company':                    '/company/about',
  '/support':                    '/company/contact',
  '/contact':                    '/company/contact',
  '/legal':                      '/legal/terms',
  '/terms':                      '/legal/terms',
  // '/logout':                  '/api/v1/account/logout',

  // Legacy (to avoid breaking links)
  '/try-fleet':                  '/get-started',

  //  ╦ ╦╔═╗╔╗ ╦ ╦╔═╗╔═╗╦╔═╔═╗
  //  ║║║║╣ ╠╩╗╠═╣║ ║║ ║╠╩╗╚═╗
  //  ╚╩╝╚═╝╚═╝╩ ╩╚═╝╚═╝╩ ╩╚═╝
  // …


  //  ╔═╗╔═╗╦  ╔═╗╔╗╔╔╦╗╔═╗╔═╗╦╔╗╔╔╦╗╔═╗
  //  ╠═╣╠═╝║  ║╣ ║║║ ║║╠═╝║ ║║║║║ ║ ╚═╗
  //  ╩ ╩╩  ╩  ╚═╝╝╚╝═╩╝╩  ╚═╝╩╝╚╝ ╩ ╚═╝
  // Note that, in this app, these API endpoints may be accessed using the `Cloud.*()` methods
  // from the Parasails library, or by using those method names as the `action` in <ajax-form>.
  // '/api/v1/account/logout':                           { action: 'account/logout' },
  // 'PUT   /api/v1/account/update-password':            { action: 'account/update-password' },
  // 'PUT   /api/v1/account/update-profile':             { action: 'account/update-profile' },
  // 'PUT   /api/v1/account/update-billing-card':        { action: 'account/update-billing-card' },
  // 'PUT   /api/v1/entrance/login':                        { action: 'entrance/login' },
  // 'POST  /api/v1/entrance/signup':                       { action: 'entrance/signup' },
  // 'POST  /api/v1/entrance/send-password-recovery-email': { action: 'entrance/send-password-recovery-email' },
  // 'POST  /api/v1/entrance/update-password-and-login':    { action: 'entrance/update-password-and-login' },
  'POST  /api/v1/deliver-contact-form-message':          { action: 'deliver-contact-form-message' },

};

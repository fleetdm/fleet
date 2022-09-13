/**
 * Policy Mappings
 * (sails.config.policies)
 *
 * Policies are simple functions which run **before** your actions.
 *
 * For more information on configuring policies, check out:
 * https://sailsjs.com/docs/concepts/policies
 */

module.exports.policies = {

  '*': 'is-logged-in',

  // Bypass the `is-logged-in` policy for:
  'entrance/*': true,
  'webhooks/*': true,
  'account/logout': true,
  'view-homepage-or-redirect': true,
  'view-faq': true,
  'view-contact': true,
  'view-get-started': true,
  'view-pricing': true,
  'legal/view-terms': true,
  'legal/view-privacy': true,
  'deliver-contact-form-message': true,
  'view-query-detail': true,
  'view-query-library': true,
  'docs/*': true,
  'handbook/*': true,
  'download-sitemap': true,
  'view-transparency': true,
  'view-press-kit': true,
  'view-platform': true,
  'view-landing': true,
  'deliver-demo-signup': true,
  'articles/*': true,
  'reports/*': true,
  'view-sales-one-pager': true,
  'try-fleet/view-register': true,
  'try-fleet/view-sandbox-login': true,
  'try-fleet/view-sandbox-teleporter-or-redirect-because-expired': true,
  'create-or-update-one-newsletter-subscription': true,
  'unsubscribe-from-all-newsletters': true,
  'view-osquery-table-details': true,

};

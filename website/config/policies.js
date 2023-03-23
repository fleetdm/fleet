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
  'admin/*': 'is-super-admin',

  // Bypass the `is-logged-in` policy for:
  'entrance/*': true,
  'webhooks/*': true,
  'imagine/*': true,
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
  'view-sales-one-pager': true,// FUTURE: Move this page into the landing-pages/ folder
  'try-fleet/view-register': true,
  'try-fleet/view-sandbox-login': true,
  'try-fleet/view-sandbox-teleporter-or-redirect-because-expired': true,
  'create-or-update-one-newsletter-subscription': true,
  'unsubscribe-from-all-newsletters': true,
  'view-osquery-table-details': true,
  'view-connect-vanta': true,
  'view-vanta-authorization': true,
  'create-vanta-authorization-request': true,
  'view-fleet-mdm': true,// FUTURE: Move this page into the landing-pages/ folder
  'deliver-mdm-beta-signup': true,
  'deliver-apple-csr': true,
  'download-rss-feed': true,
  'view-upgrade': true,// FUTURE: Move this page into the landing-pages/ folder
  'deliver-premium-upgrade-form': true,
  'view-compliance': true,// FUTURE: Move this page into the landing-pages/ folder
  'view-osquery-management': true, // FUTURE: Move this page into the landing-pages/ folder
  'experimental/*': true, // For internal Fleet demos
  'landing-pages/*': true, // For landing pages
};

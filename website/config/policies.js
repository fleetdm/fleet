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
  'query-generator/*': 'has-query-generator-access',
  'microsoft-proxy/*': 'is-cloud-customer',
  // Bypass the `is-logged-in` policy for:

  'entrance/*': true,
  'webhooks/*': true,
  'account/logout': true,
  'view-homepage-or-redirect': true,
  'view-faq': true,
  'view-contact': true,
  'view-pricing': true,
  'legal/view-terms': true,
  'legal/view-privacy': true,
  'deliver-contact-form-message': true,
  'view-query-detail': true,
  'view-policy-details': true,
  'view-query-library': true,
  'view-policy-library': true,
  'view-vital-details': true,
  'docs/*': true,
  'handbook/*': true,
  'download-sitemap': true,
  'view-transparency': true,
  'view-press-kit': true,
  'deliver-demo-signup': true,
  'articles/*': true,
  'reports/*': true,
  'try-fleet/view-sandbox-teleporter-or-redirect-because-expired-or-waitlist': true,
  'create-or-update-one-newsletter-subscription': true,
  'unsubscribe-from-all-newsletters': true,
  'unsubscribe-from-marketing-emails': true,
  'view-osquery-table-details': true,
  'view-connect-vanta': true,
  'view-vanta-authorization': true,
  'create-vanta-authorization-request': true,
  'view-device-management': true,
  'deliver-mdm-beta-signup': true,
  'deliver-apple-csr': true,
  'download-rss-feed': true,
  'view-observability': true,
  'view-software-management': true,
  'deliver-mdm-demo-email': true,
  'view-support': true,
  'view-integrations': true,
  'deliver-talk-to-us-form-submission': true,
  'get-human-interpretation-from-osquery-sql': true,
  'customers/view-new-license': true,
  'redirect-vanta-authorization-request': true,
  'view-deals': true,
  'deliver-deal-registration-submission': true,
  'get-est-device-certificate': true,
  'view-testimonials': true,
  'view-app-library': true,
  'view-app-details': true,
  'view-meetups': true,
  'view-os-settings': true,
  'view-fleetctl-preview': true,
  'get-llm-generated-configuration-profile': true,
  'account/update-start-cta-visibility': true,
  'microsoft-proxy/receive-redirect-from-microsoft': true,
  'view-configuration-builder': true,
  'android-proxy/*': true,
  'microsoft-proxy/view-remediate': true,
  'microsoft-proxy/view-turn-on-mdm': true,
};

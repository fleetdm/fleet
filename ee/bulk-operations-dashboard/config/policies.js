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
  'list-software': 'is-authorized-by-bearer-token',

  // Bypass the `is-logged-in` policy for:
  'entrance/*': true,
  'account/logout': true,
  'legal/view-terms': true,
  'legal/view-privacy': true,

};

/**
 * badConfig.js
 *
 * A custom response.
 *
 * Example usage:
 * ```
 *     return res.badConfig();
 *     // -or-
 *     return res.badConfig('builtStaticContent.queries');
 * ```
 *
 * AKA with actions2:
 * ```
 *     exits: {
 *       badConfig: { responseType: 'badConfig' }
 *     }
 * ```
 *
 * ```
 *     throw 'badConfig';
 *     // -or-
 *     throw { badConfig: 'builtStaticContent.queries' }
 * ```
 */

module.exports = function badConfig(configKeyPath) {

  let res = this.res;

  sails.log.verbose('Ran custom response: res.badConfig()');

  if (configKeyPath !== undefined && (!_.isString(configKeyPath) || configKeyPath === '' || configKeyPath.match(/^sails\.config/))) {
    throw new Error('Invalid usage of "badConfig" custom response: If specified, data sent through into the "badConfig" response should be keypath on sails.config; like "custom.internalEmailAddress", not "sails.config.custom.internalEmailAddress".  But instead, got: '+configKeyPath);
  }

  // Determine a reasonable explanation ± any further info/troubleshooting tips.
  let explanation = 'Missing, incomplete, or invalid configuration';
  if (configKeyPath === undefined) {
    explanation += `.  Please check your server logs see which action in api/controllers/ this error is coming from, find where this custom response is being called and determine which config assertion is failing, then update the relevant Sails config, and re-lift the server.`;
  } else {
    explanation += ` (sails.config.${configKeyPath}).  Please `;
    // Now for an imperative mood phrase that comes after "Please ":
    if (configKeyPath.match(/^builtStaticContent/)) {
      explanation += 'try doing `sails run build-static-content`, and then re-lifting the server.';
    } else {
      explanation += 'update this configuration, and then re-lift the server.\n [?] Unsure?  Check out: https://sailsjs.com/documentation/concepts/configuration';
    }
  }

  // Note that we don't instantiate an Error instance here because its stack trace would be cliffed out.
  return res.serverError(explanation);

};

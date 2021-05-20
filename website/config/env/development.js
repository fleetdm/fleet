/**
 * Development environment settings
 * (sails.config.*)
 *
 * The configuration in this file only applies when Sails is running
 * in the local development environment.
 */

module.exports = {

  // Develop locally on http://localhost:2024
  // (instead of the standard default for Sails apps, http://localhost:1337.
  // This helps avoid conflicts since `fleetctl preview` will very often already
  // be running on port 1337 on your computer.)
  port: 2024,

  // Add any dev-only routes for local development of not-yet-released pages.
  // e.g. http://localhost:2024/sandbox/example-query
  routes: {
    'GET /sandbox/example-query': { action: 'view-query-detail' },// « old, superceded by fancier version below (just leaving it here to avoid messing up other PR)

    'GET /sandbox/queries': { action: 'view-query-library' },// « to see it, check out /sandbox/queries
    'GET /sandbox/queries/*': { skipAssets: false, action: 'view-query-detail' },// « to see it, check out /sandbox/queries/adg
    'GET /sandbox/documentation/*': { skipAssets: false, action: 'docs/view-basic-documentation' },// « to see it, check out http://localhost:2024/sandbox/documentation/adsg
  },

};

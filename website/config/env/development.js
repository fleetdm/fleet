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

  },

};

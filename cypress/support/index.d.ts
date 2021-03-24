// load type definitions that come with Cypress module
// <reference types="cypress" />

declare namespace Cypress {
  interface Chainable {
    /**
     * Custom command to setup the testing environment.
     */
    setup(): Chainable<Element>;

    /**
     * Custom command to setup the SMTP configuration for this testing environment.
     */
    setupSMTP(): Chainable<Element>;

    /**
     * Custom command to login the user programmatically using the fleet API
     */
    login(): Chainable<Element>;
  }
}

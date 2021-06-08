// load type definitions that come with Cypress module
// <reference types="cypress" />

declare namespace Cypress {
  interface Chainable {
    /**
     * Custom command to setup the testing environment.
     */
    setup(): Chainable<Element>;

    /**
     * Custom command to login the user programmatically using the fleet API.
     */
    login(username?: string, password?: string): Chainable<Element>;

    /**
     * Custom command to log out the current user.
     */
    logout(): Chainable<Element>;

    /**
     * Custom command to add a new user in Fleet (via fleetctl).
     */
    addUser(
      username: string,
      options?: { username?: string; password?: string; globalRole?: string }
    ): Chainable<Element>;

    /**
     * Custom command to setup the SMTP configuration for this testing environment.
     *
     * NOTE: login() command is required before this, as it will make authenticated
     * requests.
     */
    setupSMTP(): Chainable<Element>;

    /**
     * Custom command to set up SSO auth with the local server.
     *
     * NOTE: login() command is required before this, as it will make authenticated
     * requests.
     */
    setupSSO(enable_idp_login?: boolean): Chainable<Element>;

    /**
     * Custom command to login a user1@example.com via SSO.
     */
    loginSSO(): Chainable<Element>;

    /**
     * Custom command to get the emails handled by the Mailhog server.
     */
    getEmails(): Chainable<Response>;

    /**
     * Custom command to seed the Core tier teams/users.
     *
     * NOTE: login() command is required before this, as it will make authenticated
     * requests.
     */
    seedCore(): Chainable<Element>;

    /**
     * Custom command to seed the Basic tier teams/users.
     *
     * NOTE: login() command is required before this, as it will make authenticated
     * requests.
     */
    seedBasic(): Chainable<Element>;

    /**
     * Custom command to seed the teams/users as represented in Figma.
     *
     * NOTE: login() command is required before this, as it will make authenticated
     * requests.
     */
    seedFigma(): Chainable<Element>;
  }
}

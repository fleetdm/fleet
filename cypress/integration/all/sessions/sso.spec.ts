import CONSTANTS from "../../../support/constants";

const { GOOD_PASSWORD } = CONSTANTS;

const enable_sso_idp_login = true;

describe("SSO Sessions", () => {
  beforeEach(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
  });
  it("non-SSO user can login with username/password", () => {
    cy.login();
    cy.setupSSO({ enable_sso_idp_login });
    cy.logout();
    cy.visit("/");
    cy.getAttached(".login-form__forgot-link").should("exist");
    // Log in
    cy.getAttached("input").first().type("admin@example.com");
    cy.getAttached("input").last().type(GOOD_PASSWORD);
    cy.contains("button", "Login").click();
    // Verify dashboard
    cy.url().should("include", "/dashboard");
    cy.contains("Hosts");
    // Log out
    cy.getAttached(".avatar").first().click();
    cy.contains("button", "Sign out").click();
    cy.url().should("match", /\/login$/);
  });
  it("can login via SSO", () => {
    cy.login();
    cy.setupSSO({ enable_sso_idp_login });
    cy.logout();
    cy.visit("/");
    // Log in
    cy.contains("button", "Sign on with SimpleSAML");
    cy.loginSSO();
    cy.contains("Hosts");
  });
  it("fails when IdP login disabled", () => {
    cy.login();
    cy.setupSSO();
    cy.logout();
    cy.visit("/");
    cy.contains("button", "Sign on with SimpleSAML");
    cy.loginSSO();
    // Log in should fail
    cy.contains("Password");
  });
  it("displays an error message when status is set", () => {
    cy.visit("/login?status=account_disabled");
    cy.getAttached(".flash-message");
  });
  it("existing users can log-in normally with JIT enabled", () => {
    cy.login();
    cy.setupSSO({
      enable_sso_idp_login: true,
      enable_jit_provisioning: true,
    });
    cy.logout();
    cy.visit("/");
    // Log in
    cy.contains("button", "Sign on with SimpleSAML");
    cy.loginSSO();
    cy.contains("Hosts");
  });
});

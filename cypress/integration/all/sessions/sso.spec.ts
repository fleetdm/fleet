describe("SSO Sessions", () => {
  beforeEach(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
  });
  it("non-SSO user can login with username/password", () => {
    cy.login();
    cy.setupSSO((enable_idp_login = true));
    cy.logout();
    cy.visit("/");
    cy.getAttached(".login-form__forgot-link").should("exist");
    // Log in
    cy.getAttached("input").first().type("admin@example.com");
    cy.getAttached("input").last().type("user123#");
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
    cy.setupSSO((enable_idp_login = true));
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
});

describe("SSO Sessions", () => {
  describe("JIT user provisioning", () => {
    before(() => {
      cy.setup();
    });

    beforeEach(() => {
      Cypress.session.clearAllSavedSessions();
    });

    it("non-existent users can't login if JIT is not enabled", () => {
      cy.login();
      cy.setupSSO({
        enable_sso_idp_login: true,
        enable_jit_provisioning: false,
      });
      cy.logout();
      cy.visit("/");
      // Log in
      cy.contains("button", "Sign on with SimpleSAML");
      cy.loginSSO({ username: "sso_user2" });
      cy.url().should("include", "/login?status=account_invalid");
      cy.getAttached(".flash-message");
    });

    it("non-existent users are provisioned if JIT is enabled", () => {
      cy.login();
      cy.setupSSO({
        enable_sso_idp_login: true,
        enable_jit_provisioning: true,
      });
      cy.logout();
      cy.visit("/");
      // Log in
      cy.contains("button", "Sign on with SimpleSAML");
      cy.loginSSO({ username: "sso_user2" });
      cy.contains("Hosts");
      cy.contains("was added to Fleet by SSO");
    });
  });
});

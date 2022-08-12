const user = {
  username: "sso_user2",
};

describe("SSO Sessions", () => {
  describe("JIT user provisioning", () => {
    before(() => {
      cy.setup();
    });

    beforeEach(() => {
      Cypress.session.clearAllSavedSessions();
    });

    it("non-existent users can't login even with JIT enabled", () => {
      cy.login();
      cy.setupSSO({
        enable_sso_idp_login: true,
        enable_jit_provisioning: true,
      });
      cy.logout();
      cy.visit("/");
      // Log in
      cy.contains("button", "Sign on with SimpleSAML");
      cy.loginSSO(user);
      cy.url().should("include", "/login?status=account_invalid");
      cy.getAttached(".flash-message");
    });
  });
});

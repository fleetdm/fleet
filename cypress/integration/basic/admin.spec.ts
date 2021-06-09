if (Cypress.env("FLEET_TIER") === "basic") {
  describe("Basic tier - Admin user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedBasic();
      cy.logout();
    });

    it("Can perform the appropriate actions", () => {
      cy.login("anna@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // TODO write the test!
    });
  });
}

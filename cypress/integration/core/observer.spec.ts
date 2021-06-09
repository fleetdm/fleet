if (Cypress.env("FLEET_TIER") === "core") {
  describe("Core tier - Observer user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedCore();
      cy.logout();
    });

    it("Can perform the appropriate actions", () => {
      cy.login("oliver@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // TODO write the test!
    });
  });
}

if (Cypress.env("FLEET_TIER") === "basic") {
  describe("Basic tier - Team observer/maintainer user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedBasic();
      cy.logout();
    });

    it("Can perform the appropriate actions", () => {
      cy.login("marco@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // TODO write the test!
    });
  });
}

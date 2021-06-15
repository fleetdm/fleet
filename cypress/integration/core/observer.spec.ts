if (Cypress.env("FLEET_TIER") === "core") {
  describe("Core tier - Observer user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedCore();
      cy.addDockerHost();
      cy.logout();
    });

    afterEach(() => {
      cy.stopDockerHost();
    });

    it("Can perform the appropriate core global observer actions", () => {
      cy.login("oliver@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // TODO write the test!
    });
  });
}

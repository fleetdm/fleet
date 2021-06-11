if (Cypress.env("FLEET_TIER") === "basic") {
  describe("Basic tier - Observer user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedBasic();
      cy.seedQueries();
      cy.logout();
    });

    it("Can perform the appropriate actions", () => {
      cy.login("oliver@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // TODO write the test!
    });

    it("Should verify Teams on Hosts page", () => {
      cy.login("marco@organization.com", "user123#");
      cy.visit("/hosts/manage");

      cy.findByText("All hosts which have enrolled in Fleet").should("exist");

      // TODO: can see the "Team" column in the Hosts table
      // cy.contains(".table-container .data-table__table th", "Team").should("be.visible");
    });

    it("Should verify hidden items on Hosts page", () => {
      cy.login("marco@organization.com", "user123#");
      cy.visit("/hosts/manage");

      cy.findByText("Packs").should("not.exist");
      cy.findByText("Queries").should("not.exist");

      // TODO: can see the "Team" column in the Hosts table
      // cy.contains(".table-container .data-table__table th", "Team").should("be.visible");
    });
  });
}

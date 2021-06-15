if (Cypress.env("FLEET_TIER") === "basic") {
  describe("Basic tier - Observer user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedBasic();
      cy.seedQueries();
      cy.addDockerHost();
      cy.logout();
    });

    afterEach(() => {
      cy.stopDockerHost();
    });

    it("Can perform the appropriate basic global observer actions", () => {
      cy.login("oliver@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // Host manage page: Can see team column
      cy.visit("/hosts/manage");

      cy.get("thead").within(() => {
        cy.findByText(/team/i).should("exist");
      });

      // Host details page: Can see team on host
      cy.get("tbody").within(() => {
        // Test host text varies
        cy.findByRole("button").click();
      });
      cy.get(".title").within(() => {
        cy.findByText("Team").should("exist");
      });

      // Query pages: Can see team in select targets dropdown
      cy.visit("/queries/manage");

      cy.findByText(/detect presence/i).click();

      cy.findByRole("button", { name: /run query/i }).click();

      cy.get(".target-select").within(() => {
        cy.findByText(/Label name, host name, IP address, etc./i).click();
        cy.findByText(/teams/i).should("exist");
      });
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

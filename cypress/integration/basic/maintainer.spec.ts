if (Cypress.env("FLEET_TIER") === "basic") {
  describe("Basic tier - Maintainer user", () => {
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

    it("Can perform the appropriate actions", () => {
      cy.login("mary@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // Host manage page: Teams column, select a team
      cy.visit("/hosts/manage");

      cy.get("thead").within(() => {
        cy.findByText(/team/i).should("exist");
      });

      cy.contains("button", /add new host/i).click();
      // TODO: Check Team Apples is in Select a team dropdown
      cy.contains("button", /done/i).click();

      // Host details page: Can see team UI
      cy.get("tbody").within(() => {
        // Test host text varies
        cy.findByRole("button").click();
      });
      cy.get(".title").within(() => {
        cy.findByText("Team").should("exist");
      });

      // Query pages: Can see teams UI for create, edit, and run query
      cy.visit("/queries/manage");

      cy.findByRole("button", { name: /create new query/i }).click();

      cy.get(".target-select").within(() => {
        cy.findByText(/Label name, host name, IP address, etc./i).click();
        cy.findByText(/teams/i).should("exist");
      });

      cy.visit("/queries/manage");

      cy.findByText(/detect presence/i).click();

      cy.findByRole("button", { name: /edit or run query/i }).click();

      cy.get(".target-select").within(() => {
        cy.findByText(/Label name, host name, IP address, etc./i).click();
        cy.findByText(/teams/i).should("exist");
      });
    });
  });
}

if (Cypress.env("FLEET_TIER") === "basic") {
  describe("Basic tier - Observer user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedBasic();
      cy.seedQueries();
      cy.addDockerHost("apples");
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

    // Pseudo code for team observer only
    // TODO: Create team observer in create_basic and build test according to new manual QA
    // it("Can perform the appropriate basic team observer only actions", () => {
    //   cy.login("TEAMOBSERVERONLY@organization.com", "user123#");
    //   cy.visit("/hosts/manage");

    //   cy.findByText("All hosts which have enrolled in Fleet").should("exist");

    //   cy.findByText("Packs").should("not.exist");
    //   cy.findByText("Settings").should("not.exist");

    //   cy.contains(".table-container .data-table__table th", "Team").should(
    //     "be.visible"
    //   );
    // });
  });
}

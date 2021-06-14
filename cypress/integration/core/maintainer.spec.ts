if (Cypress.env("FLEET_TIER") === "core") {
  describe("Core tier - Maintainer user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedCore();
      cy.logout();
    });

    it("Can perform the appropriate actions", () => {
      cy.login("mary@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // Settings and Teams restrictions
      cy.findByText(/teams/i).should("not.exist");
      cy.findByText(/settings/i).should("not.exist");
      cy.visit("/settings/organization");
      cy.findByText(/you do not have permissions/i).should("exist");

      cy.visit("/hosts/manage");

      // find way to add host
      cy.contains("button", /add new host/i).click();
      cy.findByText("select a team").should("not.exist");
      cy.contains("button", /done/i).click();

      // Can delete and create query on host
      //--click on host--
      // cy.contains("button", /delete/i).click();
      // cy.contains("button", /cancel/i).click();

      // host must be online
      // cy.contains("button", /query/i).click();
      // cy.contains("button", /create new query/i).click();

      // sent to new query page
      // cy.findByText("create new query").should("not.exist");

      cy.contains("button", /add new label/i).click();
      cy.contains("button", /cancel/i).click();

      // Can create, edit, and run query
      cy.visit("/queries/manage");
      cy.findByText(/observers can run/i).should("exist");

      cy.findByRole("button", { name: /create new query/i }).click();

      cy.findByLabelText(/query name/i)
        .click()
        .type("Query all window crashes");

      // Using class selector because third party element doesn't work with Cypress Testing Selector Library
      cy.get(".ace_scroller")
        .click({ force: true })
        .type("{selectall}{backspace}SELECT * FROM windows_crashes;");

      cy.findByLabelText(/description/i)
        .click()
        .type("See all window crashes");

      cy.findByRole("button", { name: /save/i }).click();

      cy.findByRole("button", { name: /save as new/i }).click();

      cy.get(".target-select").within(() => {
        cy.findByText(/Label name, host name, IP address, etc./i).click();
        cy.findByText(/teams/i).should("not.exist");
      });

      cy.findByRole("button", { name: /run/i }).should("exist");

      cy.visit("/queries/manage");

      cy.findByText(/query all/i).click();

      cy.findByRole("button", { name: /edit or run query/i }).click();

      // Can create, edit, delete a pack
      cy.visit("/packs/manage");

      cy.findByRole("button", { name: /create new pack/i }).click();

      cy.findByLabelText(/query pack title/i)
        .click()
        .type("Errors and crashes");

      cy.findByLabelText(/description/i)
        .click()
        .type("See all user errors and window crashes.");

      cy.findByRole("button", { name: /save query pack/i }).click();

      cy.visit("/packs/manage");

      cy.findByText(/errors and crashes/i).click();

      cy.findByText(/edit pack/i).click();

      cy.findByLabelText(/query pack title/i)
        .click()
        .type("{selectall}{backspace}Server errors");

      cy.findByLabelText(/description/i)
        .click()
        .type("{selectall}{backspace}See all server errors.");

      cy.findByRole("button", { name: /save/i }).click();

      cy.visit("/packs/manage");

      cy.get("#select-pack-1").check({ force: true });

      cy.findByRole("button", { name: /delete/i }).click();

      // Can't figure out how attach findByRole onto modal button
      // Can't use findByText because delete button under modal
      cy.get(".all-packs-page__modal-btn-wrap > .button--alert")
        .contains("button", /delete/i)
        .click();

      cy.findByText(/successfully deleted/i).should("be.visible");

      cy.findByText(/server errors/i).should("not.exist");
    });
  });
}

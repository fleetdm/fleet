describe(
  "Free tier - Maintainer user",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedFree();
      cy.seedQueries();
      cy.addDockerHost();
      cy.logout();
    });

    afterEach(() => {
      cy.stopDockerHost();
    });

    it("Can perform the appropriate core global maintainer actions", () => {
      cy.login("mary@organization.com", "user123#");
      cy.visit("/hosts/manage");

      // Ensure page is loaded
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.contains("All hosts");

      // Settings restrictions
      cy.findByText(/settings/i).should("not.exist");
      cy.visit("/settings/organization");
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByText(/you do not have permissions/i).should("exist");

      // Host manage page: No team UI, can add host and label
      cy.visit("/hosts/manage");

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByText(/teams/i).should("not.exist");
      cy.contains("button", /generate installer/i).click();
      cy.contains("button", /done/i).click();

      // See the Manage enroll secret” button. A modal appears after the user selects the button
      cy.contains("button", /manage enroll secret/i).click();
      cy.contains("button", /done/i).click();

      cy.contains("button", /add label/i).click();
      cy.contains("button", /cancel/i).click();

      // Host details page: No team UI, can delete and create new query
      cy.get("tbody").within(() => {
        // Test host text varies
        cy.findByRole("button").click();
      });
      cy.get(".title").within(() => {
        cy.findByText(/team/i).should("not.exist");
      });
      cy.contains("button", /transfer/i).should("not.exist");

      // See and select operating system
      // TODO

      // Test commented out
      // Pending fix to prevent consistent failing in GitHub

      // cy.contains("button", /delete/i)
      //   .should("exist")
      //   .click();
      // cy.contains("button", /cancel/i).click();

      // cy.contains("button", /query/i).click();
      // cy.contains("button", /create custom query/i).click();

      // Queries pages: Can create, edit, and run query
      cy.visit("/queries/manage");
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      // cy.get("thead").within(() => {
      //   cy.findByText(/observer can run/i).should("exist");
      // });

      cy.findByRole("button", { name: /create new query/i }).should("exist");

      // TODO - Fix tests according to improved query experience - MP
      // cy.findByRole("button", { name: /create new query/i }).click();

      // cy.findByLabelText(/query name/i)
      //   .click()
      //   .type("Query all window crashes");

      // cy.get(".ace_scroller")
      //   .click({ force: true })
      //   .type("{selectall}{backspace}SELECT * FROM windows_crashes;");

      // cy.findByLabelText(/description/i)
      //   .click()
      //   .type("See all window crashes");

      // cy.findByRole("button", { name: /save/i }).click();

      // cy.findByRole("button", { name: /save as new/i }).click();

      // cy.findByLabelText(/observers can run/i).click({ force: true });

      // cy.get(".target-select").within(() => {
      //   cy.findByText(/Label name, host name, IP address, etc./i).click();
      //   cy.findByText(/teams/i).should("not.exist");
      // });

      // cy.findByRole("button", { name: /run/i }).should("exist");

      // cy.visit("/queries/manage");

      // cy.findByText(/query all/i).click();

      // cy.findByText(/edit & run query/i).should("exist");

      // Packs pages: Can create, edit, delete a pack
      cy.visit("/packs/manage");
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting

      cy.findByRole("button", { name: /create new pack/i }).click();

      cy.findByLabelText(/name/i).click().type("Errors and crashes");

      cy.findByLabelText(/description/i)
        .click()
        .type("See all user errors and window crashes.");

      cy.findByRole("button", { name: /save query pack/i }).click();

      cy.visit("/packs/manage");

      cy.get(".fleet-checkbox__input").check({ force: true });

      cy.findByRole("button", { name: /delete/i }).click();

      // Can't figure out how attach findByRole onto modal button
      // Can't use findByText because delete button under modal
      cy.get(".remove-pack-modal__btn-wrap > .button--alert")
        .contains("button", /delete/i)
        .click();

      cy.findByText(/successfully deleted/i).should("be.visible");

      cy.findByText(/server errors/i).should("not.exist");

      // Schedule page: Can create, edit, remove a schedule
      // TODO: Copy flow from queryflow.spec.ts here to ensure maintainers have access

      // On the Profile page, they should…
      // See Maintainer in Role section, and no Team section
      cy.visit("/profile");
      cy.findByText(/teams/i).should("not.exist");
      cy.findByText("Role")
        .next()
        .contains(/maintainer/i);
    });
  }
);

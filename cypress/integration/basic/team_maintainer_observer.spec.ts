if (Cypress.env("FLEET_TIER") === "basic") {
  describe("Basic tier - Team observer/maintainer user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedBasic();
      cy.seedQueries();
      cy.addDockerHost();
      cy.logout();
    });

    it("Can perform the appropriate team observer actions", () => {
      cy.login("marco@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // On the Hosts page, they should…

      // See hosts
      cy.findByText(/kinda empty in here/i).should("not.exist");

      // See the “Teams” column in the Hosts table
      cy.findByRole("columnheader", { name: "Team" });

      // NOT see the “Packs” and “Settings” navigation items
      cy.findByText(/packs/i).should("not.exist");
      cy.findByText(/settings/i).should("not.exist");

      // NOT see and select the “Add new host” button // TODO confirm this should be the case for Marco, who is also a team maintainer
      // cy.findByText(/add new host/i).should("not.exist");

      // NOT see and select the “Add new label” button
      // cy.findByText(/add new label/i).should("not.exist"); // TODO confirm this should be the case for Marco, who is also a team maintainer

      // On the Host details page, they should…

      // See the “Team” information below the hostname
      cy.visit("/hosts/1");
      cy.findByText(/team/i).next().contains("Apples");

      // NOT see and select the “Delete” button
      cy.findByText(/delete/i).should("not.exist");

      // NOT see and select the “Query” button
      // cy.findByText(/query/i).should("not.exist");

      // On the Queries - manage page, they should…
      cy.findByText(/queries/i).click();
      cy.findByText(/no queries available/i).should("not.exist");

      // See and select the “Show query” button in the right side panel if the saved query has `observer_can_run` set to `false`. This button appears after the user selects a query in the Queries table.
      // TODO need seedQueries() to work

      // See and select the “Run query” button in the right side panel if the saved query has `observer_can_run` set to `true`. This button appears after the user selects a query in the Queries table.
      // TODO need seedQueries() to work

      // NOT see the “Observers can run” column in the Queries table
      // cy.findByText(/observers can run/i).should("not.exist");

      // NOT see and select the “Create new query” button // TODO confirm this should be the case for Marco
      // cy.findByText(/create new query/i).should("not.exist");

      // NOT see the “SQL” and “Packs” sections in the right side bar. These sections appear after the user selects a query in the Queries table.
      cy.get(".secondary-side-panel-container").within(() => {
        cy.findByText(/sql/i).should("not.exist");
        cy.findByText(/packs/i).should("not.exist");
      });

      // // On the Query page they should…
      // cy.visit("/queries/2"); // TODO modify once seedQueries() is finalized
      // // TODO confirm that observer only is not routed to new query page if they enter /queries/new or /queries/f1337?

      // // See the ““Show SQL” button.
      // cy.findByText(/show sql/i).click();
      // cy.findByText(/hide sql/i).should("exist");

      // // See the “Select targets” input if the saved query has `observer_can_run` set to false. // TODO confirm that this really should be true rather than false
      // cy.findByText(/select targets/i).should("exist");

      // // NOT see and edit “Query name,” “Description,” “SQL”, and “Observer can run” fields.
      // cy.findByLabelText(/query name/i).should("not.exist");
      // cy.findByLabelText(/description/i).should("not.exist");
      // cy.findByLabelText(/observers can run/i).should("not.exisit");
      // cy.get(".ace_scroller")
      //   .click({ force: true })
      //   .type("{selectall}{backspace}SELECT * FROM windows_crashes;");
      // cy.findByText(/SELECT * FROM windows_crashes;/i).should("not.exist");

      // NOT see a the “Select targets” input if the saved query has `observer_can_run` set to false.
      // cy.findByText(/select targets/i).should("not.exist");

      // NOT see a the “Teams” section in the Select target picker. This picker is summoned when the “Select targets” field is selected.
      // TODO confirm if the above is true for basic observers on multiple teams
    });

    it("Can perform the appropriate maintainer actions", () => {
      cy.login("marco@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // Ensure that the user can see and click the add new host button and then that the modal appears
      cy.findByText(/add new host/i).click();
      cy.findByText(/select a team for this new host/i).should("exist");

      // Ensure that user can see the dropdown and select a team for the new host
      cy.get(".add-host-modal__team-dropdown-wrapper").within(() => {
        cy.findByText(/select a team for this new host/i).should("exist");
        cy.get(".Select").within(() => {
          cy.findByText(/select a team/i).click();
          cy.findByText(/no team/i).should("exist");
          // cy.findByText(/apples/i).should("exist");
          // cy.findByText(/oranges/i).should("not exist");
          // ^^TODO add back these assertions after dropdown bug is fixed
        });
      });

      cy.get(".modal__ex").within(() => {
        cy.findByRole("button").click();
        cy.findByText(/select a team for this new host/i).should("not.exist");
      });

      // TODO let's add some hosts to the setup so we can confirm that this user can navigate to hosts page
      // cy.findByRole("cell").next().click();

      // Ensure that the user can create and run a new custom query
      cy.get("nav").within(() => {
        cy.findByText(/queries/i).click();
      });
      cy.findByText(/create new query/i).click();
      cy.findByText(/custom query/i).should("exist");
      cy.findByRole("button", { name: "Run" }).should("exist");
      cy.findByRole("button", { name: "Save" }).should("not.exist");

      cy.get(".ace_scroller")
        .click({ force: true })
        .type("{selectall}{backspace}SELECT * FROM windows_crashes;");

      cy.get(".target-select").within(() => {
        cy.findByText(/Label name, host name, IP address, etc./i).click();
        cy.findByText(/teams/i).should("exist");
        cy.findByText(/apples/i).should("not.exist"); // Marco is only an observer on team apples
        cy.findByText(/oranges/i) // Marco is a maintainer on team oranges
          .parent()
          .parent()
          .within(() => {
            cy.findByText(/0 hosts/i).should("exist");
            // ^^TODO modify for expected host count once hosts are seeded
          });
      });
    });
  });
}

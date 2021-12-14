describe(
  "Premium tier - Team observer/maintainer user",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedPremium();
      cy.seedQueries();
      cy.addDockerHost("apples");
      cy.addDockerHost("oranges");
      cy.logout();
    });
    afterEach(() => {
      cy.stopDockerHost();
    });

    it("Can perform the appropriate team observer actions", () => {
      cy.login("marco@organization.com", "user123#");
      cy.visit("/hosts/manage");

      // Ensure page is loaded and teams are visible
      cy.contains("Hosts");

      // On the Hosts page, they should…

      // On observing team, not see the "Generate installer" and "Manage enroll secret" buttons
      cy.contains(/apples/i);
      cy.contains("button", /generate installer/i).should("not.exist");
      cy.contains("button", /manage enroll secret/i).should("not.exist");

      // See the “Teams” column in the Hosts table
      cy.get("thead").contains(/team/i).should("exist");

      // Nav restrictions
      cy.findByText(/settings/i).should("not.exist");
      cy.findByText(/schedule/i).should("exist");
      cy.visit("/settings/organization");
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByText(/you do not have permissions/i).should("exist");
      cy.visit("/packs/manage");
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByText(/you do not have permissions/i).should("exist");

      // NOT see and select "add label"
      cy.findByRole("button", { name: /new label/i }).should("not.exist");

      // On the Host details page, they should…

      // See the “Team” information below the hostname
      // cy.visit("/hosts/1");
      // cy.findByText(/team/i).next().contains("Apples");
      // ^^TODO need new forEach script/command for admin to assign team after the host is added

      // NOT see and select the “Delete” button
      // cy.findByText(/delete/i).should("not.exist");
      // ^^ TODO this is restriction only applies to hosts where they are not a maintainer

      // NOT see and select the “Query” button
      // cy.findByText(/query/i).should("not.exist");
      // ^^ TODO this is restriction only applies to hosts where they are not a maintainer

      // On the Queries manage page, they should…
      cy.visit("/queries/manage");
      cy.findByText(/no queries available/i).should("not.exist");

      // See and select the “Show query” button in the right side panel if the saved query has `observer_can_run` set to `false`. This button appears after the user selects a query in the Queries table.
      // See and select the “Run query” button in the right side panel if the saved query has `observer_can_run` set to `true`. This button appears after the user selects a query in the Queries table.
      // ^^TODO confirm if these distinctions apply to dual-role user like Marco

      // NOT see the “Observers can run” column in the Queries table
      // cy.findByText(/observers can run/i).should("not.exist");
      // ^^TODO confirm this does not apply to dual-role user like Marco

      // NOT see and select the “Create new query” button
      // cy.findByText(/create new query/i).should("not.exist");
      // ^^TODO confirm this does not apply to dual-role user like Marco

      // NOT see the “SQL” and “Packs” sections in the right side bar. These sections appear after the user selects a query in the Queries table.
      // cy.get(".secondary-side-panel-container").within(() => {
      //   cy.findByText(/sql/i).should("not.exist");
      //   cy.findByText(/packs/i).should("not.exist");
      // });
      // ^^TODO confirm this does not apply to dual-role user like Marco

      // On the Query details page they should…
      cy.visit("/queries/1");

      // TODO - Fix tests according to improved query experience - MP
      // // See the “Show SQL” button.
      // cy.findByText(/show sql/i).click();
      // cy.findByText(/hide sql/i).should("exist");

      // // See the “Select targets” input
      // cy.findByText(/select targets/i).should("exist");

      // // NOT see and edit “Query name,” “Description,” “SQL”, and “Observer can run” fields.
      // cy.findByLabelText(/query name/i).should("not.exist");
      // cy.findByLabelText(/description/i).should("not.exist");
      // cy.findByLabelText(/observers can run/i).should("not.exist");
      // cy.get(".ace_scroller")
      //   .click({ force: true })
      //   .type("{selectall}{backspace}SELECT * FROM windows_crashes;");
      // cy.findByText(/SELECT * FROM windows_crashes;/i).should("not.exist");

      // NOT see a the “Select targets” input if the saved query has `observer_can_run` set to false.
      // cy.findByText(/select targets/i).should("not.exist");
      // ^^ TODO confirm if this restriction applies to a dual-role user like Marco

      // NOT see a the “Teams” section in the Select target picker. This picker is summoned when the “Select targets” field is selected.
      // ^^ TODO confirm if this restriction applies to a dual-role user like Marco
    });

    it("Can perform the appropriate team maintainer actions", () => {
      cy.login("marco@organization.com", "user123#");
      cy.visit("/hosts/manage");

      // Ensure page is loaded and appropriate nav links are displayed
      cy.contains("Hosts");
      cy.get("nav").within(() => {
        cy.findByText(/hosts/i).should("exist");
        cy.findByText(/queries/i).should("exist");
        cy.findByText(/schedule/i).should("exist");
        cy.findByText(/settings/i).should("not.exist");
      });

      // Ensure page is loaded and appropriate nav links are displayed
      cy.contains("Hosts");
      cy.get("nav").within(() => {
        cy.findByText(/hosts/i).should("exist");
        cy.findByText(/queries/i).should("exist");
        cy.findByText(/packs/i).should("not.exist");
        cy.findByText(/settings/i).should("not.exist");
      });

      // On the hosts page, they should…

      // See the “Teams” column in the Hosts table
      cy.get("thead").contains(/team/i).should("exist");

      // On maintaining team, see the "Generate installer" and "Manage enroll secret" buttons
      cy.visit("/hosts/manage/?team_id=2");
      cy.contains(/oranges/i);
      cy.findByRole("button", { name: /generate installer/i }).click();
      cy.findByRole("button", { name: /done/i }).click();

      // On maintaining team, add secret tests same API as edit and delete
      cy.contains("button", /manage enroll secret/i).click();
      cy.contains("button", /add secret/i).click();
      cy.contains("button", /save/i).click();
      cy.contains("button", /done/i).click();

      // On the Host details page, they should…
      // cy.visit("/hosts/1");
      // ^^TODO hosts details page returning 403 likely because we need new forEach script/command for admin to assign team after the host is added

      // See and select the “Create new query” button in the Select a query modal. This modal appears after the user selects the “Query” button
      // cy.findByRole("button", { name: /query/i }).click();
      // cy.findByRole("button", { name: /create custom query/i }).should("exist");
      // cy.get(".modal__ex").within(() => {
      //   cy.findByRole("button").click();
      // });
      // ^^TODO hosts details page returning 403 likely because we need new forEach script/command for admin to assign team after the host is added

      // TODO - Fix tests according to improved query experience - MP
      // On the Queries manage page, they should…
      // cy.visit("/queries/manage");

      // // See and select the “Create new query” button
      // cy.findByText(/create new query/i).click();
      // cy.findByText(/custom query/i).should("exist");
      // cy.findByRole("button", { name: "Run" }).should("exist");
      // cy.findByRole("button", { name: "Save" }).should("not.exist");

      // cy.get(".ace_scroller")
      //   .click({ force: true })
      //   .type("{selectall}{backspace}SELECT * FROM windows_crashes;");

      // cy.get(".target-select").within(() => {
      //   cy.findByText(/Label name, host name, IP address, etc./i).click();
      //   cy.findByText(/teams/i).should("exist");
      //   cy.findByText(/apples/i).should("not.exist"); // Marco is only an observer on team apples
      //   cy.findByText(/oranges/i) // Marco is a maintainer on team oranges
      //     .parent()
      //     .parent()
      //     .within(() => {
      //       cy.findByText(/0 hosts/i).should("exist");
      //       // ^^TODO modify for expected host count once hosts are seeded
      //     });
      // });

      // On the Schedule page, they should
      // See Oranges (team they maintain) only, not able to reach packs, able to schedule a query
      cy.visit("/schedule/manage");
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findAllByText(/oranges/i).should("exist");
      cy.findByText(/advanced/i).should("not.exist");
      cy.findByRole("button", { name: /schedule a query/i }).click();
      cy.findByText(/select query/i).click();
      cy.findByText(/detect presence/i).click();
      cy.get(".schedule-editor-modal__btn-wrap")
        .contains("button", /schedule/i)
        .click();

      cy.visit("/schedule/manage");

      cy.wait(2000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByText(/detect presence/i).should("exist");

      cy.visit("/hosts/manage");
      cy.contains(".table-container .data-table__table th", "Team").should(
        "be.visible"
      );

      // On the Profile page, they should…
      // See 2 Teams in the Team section and Various in the Role section
      cy.visit("/profile");
      cy.findByText("Teams")
        .next()
        .contains(/2 teams/i);
      cy.findByText("Role")
        .next()
        .contains(/various/i);
    });
  }
);

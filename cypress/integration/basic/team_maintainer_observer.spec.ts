describe("Basic tier - Team observer/maintainer user", () => {
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

  it("Can perform the appropriate team observer actions", () => {
    cy.login("marco@organization.com", "user123#");
    cy.visit("/");

    // Ensure page is loaded
    cy.contains("Hosts");

    // On the Hosts page, they should…

    // See hosts
    // cy.findByText(/kinda empty in here/i).should("not.exist");
    // ^^TODO hosts table is not rendering because we need new forEach script/command for admin to assign team after the host is added

    // See the “Teams” column in the Hosts table
    // cy.get("thead").contains(/team/i).should("exist");

    // Nav restrictions
    cy.findByText(/settings/i).should("not.exist");
    cy.findByText(/schedule/i).should("not.exist");
    cy.visit("/settings/organization");
    cy.findByText(/you do not have permissions/i).should("exist");
    cy.visit("/packs/manage");
    cy.findByText(/you do not have permissions/i).should("exist");
    cy.visit("/schedule/manage");
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

    // See the “Show SQL” button.
    cy.findByText(/show sql/i).click();
    cy.findByText(/hide sql/i).should("exist");

    // See the “Select targets” input
    cy.findByText(/select targets/i).should("exist");

    // NOT see and edit “Query name,” “Description,” “SQL”, and “Observer can run” fields.
    cy.findByLabelText(/query name/i).should("not.exist");
    cy.findByLabelText(/description/i).should("not.exist");
    cy.findByLabelText(/observers can run/i).should("not.exist");
    cy.get(".ace_scroller")
      .click({ force: true })
      .type("{selectall}{backspace}SELECT * FROM windows_crashes;");
    cy.findByText(/SELECT * FROM windows_crashes;/i).should("not.exist");

    // NOT see a the “Select targets” input if the saved query has `observer_can_run` set to false.
    // cy.findByText(/select targets/i).should("not.exist");
    // ^^ TODO confirm if this restriction applies to a dual-role user like Marco

    // NOT see a the “Teams” section in the Select target picker. This picker is summoned when the “Select targets” field is selected.
    // ^^ TODO confirm if this restriction applies to a dual-role user like Marco
  });

  it("Can perform the appropriate maintainer actions", () => {
    cy.login("marco@organization.com", "user123#");
    cy.visit("/");

    // Ensure page is loaded and appropriate nav links are displayed
    cy.contains("Hosts");
    cy.get("nav").within(() => {
      cy.findByText(/hosts/i).should("exist");
      cy.findByText(/queries/i).should("exist");
      cy.findByText(/schedule/i).should("not.exist");
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
    // cy.get("thead").contains(/team/i).should("exist");
    // ^^TODO hosts table is not rendering because we need new forEach script/command for admin to assign team after the host is added

    // See and select the “Add new host” button
    cy.findByText(/add new host/i).click();

    // See the “Select a team for this new host” in the Add new host modal. This modal appears after the user selects the “Add new host” button
    cy.get(".add-host-modal__team-dropdown-wrapper").within(() => {
      cy.findByText(/select a team for this new host/i).should("exist");
      cy.get(".Select").within(() => {
        cy.findByText(/select a team/i).click();
        cy.findByText(/no team/i).should("exist");
        // cy.findByText(/apples/i).should("exist");
        // cy.findByText(/oranges/i).should("not exist");
        // ^ TODO: Team maintainer has access to only their teams, team observer does not have access
      });
    });
    cy.findByRole("button", { name: /done/i }).click();

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

    // On the Queries manage page, they should…
    cy.visit("/queries/manage");

    // See and select the “Create new query” button
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
});

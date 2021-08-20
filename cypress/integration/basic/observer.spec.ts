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
    cy.contains("button", /transfer/i).should("not.exist");

    // Query pages: Can see team in select targets dropdown
    cy.visit("/queries/manage");

    cy.findByText(/detect presence/i).click();

    cy.findByRole("button", { name: /run/i }).click();

    cy.get(".target-select").within(() => {
      cy.findByText(/Label name, host name, IP address, etc./i).click();
      cy.findByText(/teams/i).should("exist");
    });
  });

  // Pseudo code for team observer only
  // TODO: Rebuild this test according to new manual QA
  it("Can perform the appropriate basic team observer only actions", () => {
    cy.login("toni@organization.com", "user123#");
    cy.visit("/hosts/manage");

    cy.findByText("Hosts").should("exist");

    // Nav restrictions
    cy.findByText(/settings/i).should("not.exist");
    cy.findByText(/schedule/i).should("not.exist");
    cy.visit("/settings/organization");
    cy.findByText(/you do not have permissions/i).should("exist");
    cy.visit("/packs/manage");
    cy.findByText(/you do not have permissions/i).should("exist");
    cy.visit("/schedule/manage");
    cy.findByText(/you do not have permissions/i).should("exist");

    cy.contains(".table-container .data-table__table th", "Team").should(
      "be.visible"
    );

    // On the Profile page, they should…
    // See Global in the Team section and Observer in the Role section
    cy.visit("/profile");
    cy.findByText(/team/i)
      .next()
      .contains(/apples/i);
    cy.findByText("Role")
      .next()
      .contains(/observer/i);
  });
});

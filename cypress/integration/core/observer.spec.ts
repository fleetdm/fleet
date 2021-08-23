describe("Core tier - Observer user", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.seedCore();
    cy.seedQueries();
    cy.addDockerHost();
    cy.logout();
  });

  afterEach(() => {
    cy.stopDockerHost();
  });

  it("Can perform the appropriate core global observer actions", () => {
    cy.login("oliver@organization.com", "user123#");
    cy.visit("/");

    // Ensure page is loaded
    cy.contains("All hosts");

    // Nav restrictions
    cy.findByText(/settings/i).should("not.exist");
    cy.findByText(/schedule/i).should("not.exist");
    cy.visit("/settings/organization");
    cy.findByText(/you do not have permissions/i).should("exist");
    cy.visit("/packs/manage");
    cy.findByText(/you do not have permissions/i).should("exist");
    cy.visit("/schedule/manage");
    cy.findByText(/you do not have permissions/i).should("exist");

    // Host manage page: No team UI, cannot add host or label
    cy.visit("/hosts/manage");
    cy.findByText(/teams/i).should("not.exist");
    cy.contains("button", /add new host/i).should("not.exist");
    cy.contains("button", /add label/i).should("not.exist");

    // Host details page: No team UI, cannot delete or query
    cy.get("tbody").within(() => {
      // Test host text varies
      cy.findByRole("button").click();
    });

    cy.findByText(/team/i).should("not.exist");
    cy.contains("button", /transfer/i).should("not.exist");
    cy.contains("button", /delete/i).should("not.exist");
    cy.contains("button", /query/i).should("not.exist");

    // Queries pages: Observer can or cannot run UI
    cy.visit("/queries/manage");
    cy.get("thead").within(() => {
      cy.findByText(/observer can run/i).should("not.exist");
    });

    cy.findByRole("button", { name: /create new query/i }).should("not.exist");

    cy.findByText(/detect presence/i).click();
    cy.findByText(/packs/i).should("not.exist");
    cy.findByLabelText(/query name/i).should("not.exist");
    cy.findByLabelText(/sql/i).should("not.exist");
    cy.findByLabelText(/description/i).should("not.exist");
    cy.findByLabelText(/observer can run/i).should("not.exist");
    cy.findByText(/show sql/i).click();
    cy.get(".target-select").within(() => {
      cy.findByText(/Label name, host name, IP address, etc./i).click();
      cy.findByText(/teams/i).should("not.exist");
    });
    cy.findByRole("button", { name: /run/i }).should("exist");

    cy.visit("/queries/manage");

    cy.findByText(/get authorized/i).click();
    cy.findByText(/packs/i).should("not.exist");
    cy.findByLabelText(/query name/i).should("not.exist");
    cy.findByLabelText(/sql/i).should("not.exist");
    cy.findByLabelText(/description/i).should("not.exist");
    cy.findByLabelText(/observer can run/i).should("not.exist");
    cy.findByText(/show sql/i).click();
    cy.get(".target-select").should("not.exist");
    cy.findByRole("button", { name: /run/i }).should("not.exist");

    // On the Profile page, they should…
    // See Observer in Role section, and no Team section
    cy.visit("/profile");
    cy.findByText(/teams/i).should("not.exist");
    cy.findByText("Role")
      .next()
      .contains(/observer/i);
  });
});

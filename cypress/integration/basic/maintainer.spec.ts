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

  it("Can perform the appropriate basic global maintainer actions", () => {
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
    cy.contains("button", /transfer/i).click();
    cy.get(".Select-control").click();
    cy.findByText(/create a team/i).should("not.exist");
    cy.get(".Select-menu").within(() => {
      cy.findByText(/no team/i).should("exist");
      cy.findByText(/apples/i).should("exist");
      cy.findByText(/oranges/i).click();
    });
    cy.get(".transfer-action-btn").click();
    cy.findByText(/transferred to oranges/i).should("exist");
    cy.findByText(/team/i).next().contains("Oranges");

    // Query pages: Can see teams UI for create, edit, and run query
    cy.visit("/queries/manage");

    cy.findByRole("button", { name: /create new query/i }).click();

    cy.get(".target-select").within(() => {
      cy.findByText(/Label name, host name, IP address, etc./i).click();
      cy.findByText(/teams/i).should("exist");
    });

    cy.visit("/queries/manage");

    cy.findByText(/detect presence/i).click();

    cy.findByText(/edit & run query/i).should("exist");

    cy.get(".target-select").within(() => {
      cy.findByText(/Label name, host name, IP address, etc./i).click();
      cy.findByText(/teams/i).should("exist");
    });

    // On the Packs pages (manage, new, and edit), they should…
    // On the Schedule pages (manage, new, and edit), they should…
    // ^^General maintainer functionality for packs page is being tested in core/maintainer.spec.ts

    // On the Profile page, they should…
    // See Global in the Team section and Maintainer in the Role section
    cy.visit("/profile");
    cy.findByText(/team/i)
      .next()
      .contains(/global/i);
    cy.findByText("Role")
      .next()
      .contains(/maintainer/i);
  });
});

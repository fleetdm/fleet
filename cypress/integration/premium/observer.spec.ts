describe("Premium tier - Observer user", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.seedPremium();
    cy.seedQueries();
    cy.seedPolicies("apples");
    cy.addDockerHost("apples");
    cy.logout();
  });

  afterEach(() => {
    cy.stopDockerHost();
  });

  it("Can perform the appropriate basic global observer actions", () => {
    cy.login("oliver@organization.com", "user123#");
    // Host manage page: Can see team column
    cy.visit("/hosts/manage");

    // Ensure page is loaded
    cy.getAttached(".Select-value-label").contains("All teams");

    // Not see the "Manage enroll secret” or "Generate installer" button
    cy.contains("button", /manage enroll secret/i).should("not.exist");
    cy.contains("button", /generate installer/i).should("not.exist");

    cy.getAttached("thead").within(() => {
      cy.findByText(/team/i).should("exist");
    });

    // Navigate to host details page
    cy.getAttached("tbody").within(() => {
      // Test host text varies
      cy.findByRole("button").click();
    });

    // Click query button and confirm observer cannot create custom query
    cy.getAttached(".host-details__query-button").click();
    cy.contains("button", /create custom query/i).should("not.exist");
    cy.getAttached(".modal__ex").click();

    // Confirm other actions are not available to observer
    cy.getAttached(".host-details__action-button-container").within(() => {
      cy.contains("button", /transfer/i).should("not.exist");
      cy.contains("button", /delete/i).should("not.exist");
    });

    // Confirm additional host details for observer
    cy.getAttached(".info-flex").within(() => {
      // Team is shown for host
      cy.findByText(/apples/i).should("exist");
      // OS is shown for host
      cy.findByText(/ubuntu/i).should("exist");
      // Observer cannot create a new OS policy
      cy.findByRole("button").should("not.exist");
    });

    // Query pages: Can see team in select targets dropdown
    cy.visit("/queries/manage");

    cy.getAttached("tbody").within(() => {
      cy.getAttached("tr")
        .first()
        .within(() => {
          cy.contains(".fleet-checkbox__input").should("not.exist");
          cy.findByText(/detect presence/i).click();
        });
    });

    cy.getAttached(".query-form__button-wrap").within(() => {
      cy.findByRole("button", { name: /run/i }).click();
    });

    cy.contains("h3", /teams/i).should("exist");
    cy.contains(".selector-name", /apples/i).should("exist");

    // Navigate to manage policies page
    cy.contains("a", "Policies").click();
    // Not see the "Manage automations" button
    cy.findByRole("button", { name: /manage automations/i }).should(
      "not.exist"
    );

    // Cannot see and select the "Add a policy", "delete", and "edit" policy
    cy.findByRole("button", { name: /add a policy/i }).should("not.exist");

    // No global policies seeded, switch to team apples to ensure cannot create, delete, edit
    cy.findByText(/ask yes or no questions/i).should("exist");
    cy.getAttached(".Select-control").within(() => {
      cy.findByText(/all teams/i).click();
    });
    cy.getAttached(".Select-menu")
      .contains(/apples/i)
      .click();

    // Not see the "Add a policy", "delete", "save", "run" policy
    cy.findByRole("button", { name: /add a policy/i }).should("not.exist");

    cy.getAttached("tbody").within(() => {
      cy.getAttached("tr")
        .first()
        .within(() => {
          cy.contains(".fleet-checkbox__input").should("not.exist");
          cy.findByText(/filevault enabled/i).click();
        });
    });
    cy.getAttached(".policy-form__wrapper").within(() => {
      cy.findByRole("button", { name: /run/i }).should("not.exist");
      cy.findByRole("button", { name: /save/i }).should("not.exist");
    });
  });

  it("Can perform the appropriate basic team observer only actions", () => {
    cy.login("toni@organization.com", "user123#");
    cy.visit("/hosts/manage");

    // Ensure the page is loaded and teams are visible
    cy.getAttached(".data-table__table th")
      .contains("Team")
      .should("be.visible");

    // Nav restrictions
    cy.findByText(/settings/i).should("not.exist");
    cy.findByText(/schedule/i).should("not.exist");
    cy.visit("/settings/organization");
    cy.findByText(/you do not have permissions/i).should("exist");
    cy.visit("/packs/manage");
    cy.findByText(/you do not have permissions/i).should("exist");
    cy.visit("/schedule/manage");
    cy.findByText(/you do not have permissions/i).should("exist");

    // On the policies manage page, they should…
    cy.visit("/policies/manage");

    // Not see and select the "Add a policy", "delete", and "edit" policy
    cy.findByRole("button", { name: /add a policy/i }).should("not.exist");
    cy.findByText(/all teams/i).should("not.exist");
    cy.getAttached("tbody").within(() => {
      cy.getAttached("tr")
        .first()
        .within(() => {
          cy.contains(".fleet-checkbox__input").should("not.exist");
          cy.findByText(/filevault enabled/i).click();
        });
    });

    cy.getAttached(".policy-form__wrapper").within(() => {
      cy.findByRole("button", { name: /run/i }).should("not.exist");
      cy.findByRole("button", { name: /save/i }).should("not.exist");
    });

    // On the Profile page, they should…
    // See Global in the Team section and Observer in the Role section
    cy.visit("/profile");

    cy.getAttached(".user-settings__additional").within(() => {
      cy.findByText(/team/i)
        .next()
        .contains(/apples/i);
      cy.findByText("Role")
        .next()
        .contains(/observer/i);
    });
  });
});

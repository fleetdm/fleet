describe("Premium tier - Observer user", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedPremium();
    cy.seedQueries();
    cy.seedPolicies("apples");
    cy.addDockerHost("apples");
  });

  after(() => {
    cy.logout();
    cy.stopDockerHost();
  });

  describe("Global observer", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", "user123#");
    });
    describe("Manage hosts page", () => {
      beforeEach(() => cy.visit("/hosts/manage"));
      it("should render elements according to role-based access controls", () => {
        // Ensure page is loaded with teams dropdown
        cy.getAttached(".Select-value-label").contains("All teams");
        // Not see the "Manage enroll secret” or "Generate installer" button
        cy.contains("button", /manage enroll secret/i).should("not.exist");
        cy.contains("button", /generate installer/i).should("not.exist");
        // Hosts table includes teams column
        cy.getAttached("thead").within(() => {
          cy.findByText(/team/i).should("exist");
        });
      });
    });
    describe("Host details page", () => {
      beforeEach(() => cy.visit("/hosts/manage"));
      it("should render elements according to role-based access controls", () => {
        // Navigate to host details page for first host
        cy.getAttached(".hostname__cell").first().click();

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
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => cy.visit("/software/manage"));
      it("hides manage automations button", () => {
        cy.getAttached(".manage-software-page__header-wrap").within(() => {
          cy.findByRole("button", { name: /manage automations/i }).should(
            "not.exist"
          );
        });
      });
    });
    describe("Query pages", () => {
      beforeEach(() => cy.visit("/queries/manage"));
      it("should render elements according to role-based access controls", () => {
        // Navigate to query detail page for first query on manage queries page
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
      });
    });
    describe("Policies pages", () => {
      beforeEach(() => cy.visit("/policies/manage"));
      it("should render elements according to role-based access controls", () => {
        // No global policies seeded, placeholder displayed
        cy.findByText(/ask yes or no questions/i).should("exist");
        cy.findByText(/all your hosts/i).should("exist");

        // Cannot see "Manage automations" button
        cy.findByRole("button", { name: /manage automations/i }).should(
          "not.exist"
        );
        // Cannot see "Add a policy" button
        cy.findByRole("button", { name: /add a policy/i }).should("not.exist");

        // Switch to team policies
        cy.getAttached(".Select-control").within(() => {
          cy.findByText(/all teams/i).click();
        });
        cy.getAttached(".Select-menu")
          .contains(/apples/i)
          .click();
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
    });
  });

  describe("Team observer", () => {
    beforeEach(() => {
      cy.loginWithCySession("toni@organization.com", "user123#");
    });
    describe("Nav restrictions", () => {
      it("should restrict navigation according to role-based access controls", () => {
        // cypress tends to fail on uncaught exceptions. since we have
        // our own error handling, it's suggested to use this block to
        // suppress so the tests will keep running
        Cypress.on("uncaught:exception", () => {
          return false;
        });
        cy.findByText(/settings/i).should("not.exist");
        cy.findByText(/schedule/i).should("not.exist");
        cy.visit("/settings/organization");
        cy.findByText(/you do not have permissions/i).should("exist");
        cy.visit("/packs/manage");
        cy.findByText(/you do not have permissions/i).should("exist");
        cy.visit("/schedule/manage");
        cy.findByText(/you do not have permissions/i).should("exist");
      });
    });
    describe("Manage hosts page", () => {
      it("should render elements according to role-based access controls", () => {
        cy.visit("/hosts/manage");
        // Hosts table includes teams column
        cy.getAttached(".data-table__table th")
          .contains("Team")
          .should("be.visible");
      });
    });
    describe("Manage policies page", () => {
      it("should render elements according to role-based access controls", () => {
        cy.visit("/policies/manage");
        cy.findByRole("button", { name: /add a policy/i }).should("not.exist");
        cy.findByText(/all teams/i).should("not.exist");
      });
    });
    describe("Policy detail page", () => {
      it("should render elements according to role-based access controls", () => {
        cy.visit("/policies/manage");
        // Navigate to policy detail page for first policy in manage policies table
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
    });
    describe("User profile page", () => {
      it("should render elements according to role-based access controls", () => {
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
  });
});

describe("Premium tier - Maintainer user", () => {
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

  describe("Global maintainer", () => {
    beforeEach(() => {
      cy.loginWithCySession("mary@organization.com", "user123#");
    });
    describe("Manage hosts page", () => {
      it("renders elements according to role-based access controls", () => {
        cy.visit("/hosts/manage");
        // Hosts table includes teams column
        cy.getAttached(".data-table__table th")
          .contains("Team")
          .should("be.visible");
        cy.getAttached(".button-wrap")
          .contains("button", /generate installer/i)
          .click();
        cy.getAttached(".modal__content").contains("button", /done/i).click();

        cy.getAttached(".button-wrap")
          .contains("button", /manage enroll secret/i)
          .click();
        cy.getAttached(".enroll-secret-modal__add-secret")
          .contains("button", /add secret/i)
          .click();
        cy.getAttached(".secret-editor-modal__button-wrap")
          .contains("button", /save/i)
          .click();
        cy.getAttached(".enroll-secret-modal__button-wrap")
          .contains("button", /done/i)
          .click();
      });
    });
    describe("Host details page", () => {
      beforeEach(() => {
        cy.visit("/hosts/manage");
        cy.getAttached(".hostname__cell").first().click();
      });
      it("allows global maintainer to transfer host to an existing team", () => {
        cy.getAttached(".host-details__transfer-button").click();
        cy.findByText(/create a team/i).should("not.exist");
        cy.getAttached(".Select-control").click();
        cy.getAttached(".Select-menu").within(() => {
          cy.findByText(/no team/i).should("exist");
          cy.findByText(/apples/i).should("exist");
          cy.findByText(/oranges/i).click();
        });
        cy.getAttached(".transfer-action-btn").click();
        cy.findByText(/transferred to oranges/i).should("exist");
        cy.findByText(/team/i).next().contains("Oranges");
      });
      it("allows global maintainer to create an operating system policy", () => {
        cy.getAttached(".info-flex").within(() => {
          // OS is shown for host
          cy.findByText(/ubuntu/i).should("exist");
          // Observer cannot create a new OS policy
          cy.getAttached(".host-details__os-policy-button").click();
        });
        cy.getAttached(".modal__content")
          .findByRole("button", { name: /create new policy/i })
          .should("exist");
      });
      it("allows global maintainer to create a custom query", () => {
        // Click query button and confirm maintainer can see "create custom query" button
        cy.getAttached(".host-details__query-button").click();
        cy.contains("button", /create custom query/i).should("exist");
        cy.getAttached(".modal__ex").click();
      });
      it("allows global maintainer to delete a host", () => {
        cy.getAttached(".host-details__action-button-container")
          .contains("button", /delete/i)
          .click();
        cy.getAttached(".host-details__modal").within(() => {
          cy.findByText(/delete host/i).should("exist");
          cy.contains("button", /delete/i).should("exist");
          cy.getAttached(".modal__ex").click();
        });
      });
    });
    describe("Manage policies page", () => {
      beforeEach(() => cy.visit("/policies/manage"));
      it("allows global maintainer to click 'Manage automations' button", () => {
        cy.getAttached(".button-wrap")
          .findByRole("button", { name: /manage automations/i })
          .click();
        cy.findByRole("button", { name: /cancel/i }).click();
      });
      it("allows global maintainer to add a new policy", () => {
        cy.getAttached(".button-wrap")
          .findByRole("button", { name: /add a polic/i })
          .click();
        // Add a default policy
        cy.findByText(/gatekeeper enabled/i).click();
        cy.getAttached(".policy-form__button-wrap--new-policy").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save policy/i }).click();
        });
        cy.findByRole("button", { name: /^Save$/ }).click();
        cy.findByText(/policy created/i).should("exist");
      });
    });
    it("allows global maintainer to delete a team policy", () => {
      cy.visit("/policies/manage");
      cy.getAttached(".Select-control").within(() => {
        cy.findByText(/all teams/i).click();
      });
      cy.getAttached(".Select-menu")
        .contains(/apples/i)
        .click();
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.getAttached(".fleet-checkbox__input").check({ force: true });
          });
      });
      cy.findByRole("button", { name: /delete/i }).click();
      cy.getAttached(".remove-policies-modal").within(() => {
        cy.findByRole("button", { name: /delete/i }).should("exist");
        cy.findByRole("button", { name: /cancel/i }).click();
      });
    });
    it("allows global maintainer to edit a team policy", () => {
      cy.visit("policies/manage");
      cy.findByText(/all teams/i).click();
      cy.findByText(/apples/i).click();
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.getAttached(".fleet-checkbox__input").check({ force: true });
          });
      });
      cy.findByText(/filevault enabled/i).click();
      cy.getAttached(".policy-form__button-wrap").within(() => {
        cy.findByRole("button", { name: /run/i }).should("exist");
        cy.findByRole("button", { name: /save/i }).should("exist");
      });
    });
    describe("User profile page", () => {
      it("renders elements according to role-based access controls", () => {
        cy.visit("/profile");
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText(/team/i)
            .next()
            .contains(/global/i);
          cy.findByText("Role")
            .next()
            .contains(/maintainer/i);
        });
      });
    });
  });
});

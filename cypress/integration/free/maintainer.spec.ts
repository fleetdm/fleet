describe(
  "Free tier - Maintainer user",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    before(() => {
      Cypress.session.clearAllSavedSessions();
      cy.setup();
      cy.loginWithCySession();
      cy.setupSMTP();
      cy.seedFree();
      cy.seedQueries();
      cy.seedPolicies();
      cy.addDockerHost();
    });

    after(() => {
      cy.logout();
      cy.stopDockerHost();
    });

    describe("Mange hosts tests", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/hosts/manage");
      });

      it("Can verify user is on the Manage Hosts page", () => {
        cy.getAttached(".manage-hosts").within(() => {
          cy.findByText(/edit columns/i).should("exist");
        });
      });

      it("Can verify teams is disabled", () => {
        cy.contains(/team/i).should("not.exist");
      });

      it("Can see and click the 'Generate installer' button", () => {
        cy.findByRole("button", { name: /generate installer/i }).click();
        cy.contains(/team/i).should("not.exist");
        cy.contains("button", /done/i).click();
      });

      it("Can manage the enroll secret", () => {
        cy.contains("button", /manage enroll secret/i).click();
        cy.contains("button", /done/i).click();
      });

      it("Can open the 'Add label' form", () => {
        cy.findByRole("button", { name: /add label/i }).click();
        cy.findByRole("button", { name: /cancel/i }).click();
      });
    });

    describe("Host details tests", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/hosts/1");
      });

      it("Can verify teams is disabled", () => {
        cy.findByText(/team/i).should("not.exist");
        cy.contains("button", /transfer/i).should("not.exist");
      });

      it("Can delete a query", () => {
        cy.findByRole("button", { name: /delete/i }).click();
        cy.findByText(/delete host/i).should("exist");
        cy.findByRole("button", { name: /cancel/i }).click();
      });

      it("Can create a new query", () => {
        cy.findByRole("button", { name: /query/i }).click();
        cy.findByRole("button", { name: /create custom query/i }).should(
          "exist"
        );
        cy.getAttached(".modal__ex").within(() => {
          cy.findByRole("button").click();
        });
      });
    });

    describe("Queries tests", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/queries/manage");
      });

      it("Can see the 'Observer can run' column on the queries table", () => {
        cy.contains(/observer can run/i);
      });

      it("Can add a new query", () => {
        cy.findByRole("button", { name: /new query/i }).click();
        cy.getAttached(".ace_text-input")
          .first()
          .click({ force: true })
          .type("{selectall}{backspace}SELECT * FROM cypress;", {
            force: true,
          });
        cy.findByRole("button", { name: /save/i }).click();
        cy.findByLabelText(/name/i).click().type("Cypress test query");
        cy.findByLabelText(/description/i)
          .click()
          .type("Cypress test of create new query flow.");
        cy.findByLabelText(/observers can run/i).click({ force: true });
        cy.findByRole("button", { name: /save query/i }).click();
        cy.findByText(/query created/i).should("exist");
      });

      it("Can edit a query", () => {
        cy.findByText(/cypress test query/i).click({ force: true });
        cy.getAttached(".ace_text-input")
          .first()
          .click({ force: true })
          .type("{selectall}{backspace}SELECT 1 FROM cypress;", {
            force: true,
          });
        cy.findByText("Save").click(); // we have 'save as new' also
        cy.findByText(/query updated/i).should("exist");
      });

      it("Can run a query", () => {
        cy.findByText(/cypress test query/i).click({ force: true });
        cy.findByText(/run query/i).click({ force: true });
        cy.findByText(/select targets/i).should("exist");
        cy.findByText(/all hosts/i).click();
        cy.findByText(/targets selected/i).should("exist"); // target count
        cy.findByText(/run/i).click();
        cy.findByText(/querying selected hosts/i).should("exist"); // target count
      });
    });

    describe("Policies tests", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/policies/manage");
      });

      it("Can manage automations", () => {
        cy.findByRole("button", { name: /manage automations/i }).click();
        cy.findByRole("button", { name: /cancel/i }).click();
      });

      it("Can add a policy", () => {
        cy.findByRole("button", { name: /add a policy/i }).click();
        cy.getAttached(".modal__ex").within(() => {
          cy.findByRole("button").click();
        });
      });

      it("Can delete a policy", () => {
        // select checkmark on table
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

      it("Can select a policy and verify user can run and save", () => {
        cy.getAttached(".data-table__table").within(() => {
          cy.getAttached("tbody").within(() => {
            cy.getAttached("tr")
              .first()
              .within(() => {
              cy.findByRole("button", { name: /filevault enabled/i }).click();
            });
          });
        });
        cy.getAttached(".policy-form__button-wrap--new-policy").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save/i }).should("exist");
        });
      });
    });

    describe("Packs tests", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/packs/manage");
      });

      it("Can create a pack", () => {
        cy.findByRole("button", { name: /create new pack/i }).click();
        cy.findByLabelText(/name/i).click().type("Errors and crashes");
        cy.findByLabelText(/description/i)
          .click()
          .type("See all user errors and window crashes.");
        cy.findByRole("button", { name: /save query pack/i }).click();
      });

      it("Can delete a pack", () => {
        // select checkmark on table
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.getAttached(".fleet-checkbox__input").check({ force: true });
            });
        });
        // cy.get(".fleet-checkbox__input").check({ force: true });
        cy.findByRole("button", { name: /delete/i }).click();

        // Can't figure out how attach findByRole onto modal button
        // Can't use findByText because delete button under modal
        cy.get(".remove-pack-modal__btn-wrap > .button--alert")
          .contains("button", /delete/i)
          .click();

        cy.findByText(/successfully deleted/i).should("be.visible");
        cy.findByText(/server errors/i).should("not.exist");
      });
    });

    describe("Profile tests", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/profile");
      });

      it("Can verify teams is disabled for the Profile page", () => {
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText(/teams/i).should("not.exist");
        });
      });

      it("Can verify the role of the user is maintainer", () => {
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText("Role")
            .next()
            .contains(/maintainer/i);
        });
      });
    });

    // nav restrictions are at the end because we expect to see a
    // 403 error overlay which will hide the nav and make the test fail
    describe("Nav restrictions", () => {
      // cypress tends to fail on uncaught exceptions. since we have
      // our own error handling, it's suggested to use this block to
      // suppress so the tests will keep running
      Cypress.on("uncaught:exception", () => {
        return false;
      });
      
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
      });

      it("Can verify user does not have access to settings", () => {
        cy.findByText(/settings/i).should("not.exist");
        cy.visit("/settings/organization");
        cy.findByText(/you do not have permissions/i).should("exist");
      });
    });
  }
);

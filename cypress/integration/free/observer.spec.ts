describe("Free tier - Observer user", () => {
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
      cy.loginWithCySession("oliver@organization.com", "user123#");
      cy.visit("/hosts/manage");
    });

    it("Can verify nav is restricted", () => {
      // we expect a 402 error from the teams API
      // in Cypress, we can't update the context for if we're
      // in the premium tier, so the tests runs the teams API
      Cypress.on("uncaught:exception", () => {
        return false;
      });

      // Nav restrictions
      cy.findByText(/settings/i).should("not.exist");
      cy.findByText(/schedule/i).should("not.exist");
      cy.visit("/settings/organization");
      cy.findByText(/you do not have permissions/i).should("exist");
      cy.visit("/packs/manage");
      cy.findByText(/you do not have permissions/i).should("exist");
      cy.visit("/schedule/manage");
      cy.findByText(/you do not have permissions/i).should("exist");
    });

    it("Can verify teams is disabled", () => {
      cy.findByText(/teams/i).should("not.exist");
    });

    it("Can verify user cannot generate an installer", () => {
      cy.contains("button", /generate installer/i).should("not.exist");
    });

    it("Can verify user cannot add a label", () => {
      cy.contains("button", /add label/i).should("not.exist");
    });

    it("Can verify user cannot manage the enroll secret", () => {
      cy.contains("button", /manage enroll secret/i).should("not.exist");
    });
  });

  describe("Host details tests", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", "user123#");
      cy.visit("/hosts/1");
    });

    it("Can verify teams is disabled", () => {
      cy.findByText(/team/i).should("not.exist");
    });

    it("Can verify user cannot transfer host", () => {
      cy.contains("button", /transfer/i).should("not.exist");
    });

    it("Can verify user cannot delete host", () => {
      cy.contains("button", /delete/i).should("not.exist");
    });

    it("Can verify user cannot query host", () => {
      cy.contains("button", /query/i).click();
    });

    it("Can verify user cannot create query", () => {
      cy.contains("button", /create custom query/i).should("not.exist");
    });
  });

  describe("Queries tests", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", "user123#");
      cy.visit("/queries/manage");
    });

    it("Can verify that user does not see 'Observer can run' column", () => {
      cy.getAttached("thead").within(() => {
        cy.findByText(/observer can run/i).should("not.exist");
      });
    });

    it("Can verify that user cannot create a query", () => {
      cy.findByRole("button", { name: /create new query/i }).should(
        "not.exist"
      );
    });

    it("Can verify that user can select a query and only run it", () => {
      cy.getAttached(".data-table__table").within(() => {
        cy.findByRole("button", { name: /detect presence/i }).click();
      });
      cy.findByText(/packs/i).should("not.exist");
      cy.findByLabelText(/query name/i).should("not.exist");
      cy.findByLabelText(/sql/i).should("not.exist");
      cy.findByLabelText(/description/i).should("not.exist");
      cy.findByLabelText(/observer can run/i).should("not.exist");
      cy.findByText(/show sql/i).click();
      cy.findByRole("button", { name: /run query/i }).should("exist");
    });
  });

  describe("Policies tests", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", "user123#");
      cy.visit("/policies/manage");
    });

    it("Can verify user cannot manage automations", () => {
      cy.findByRole("button", { name: /manage automations/i }).should(
        "not.exist"
      );
    });

    it("Can verify user cannot add a policy", () => {
      cy.findByRole("button", { name: /add a policy/i }).should("not.exist");
    });

    it("Can verify user cannot run, edit, or delete a policy", () => {
      cy.getAttached("tbody").within(() => {
        cy.get("tr")
          .first()
          .within(() => {
            cy.get(".fleet-checkbox__input").should("not.exist");
          });
        cy.findByText(/filevault enabled/i).click();
      });

      cy.getAttached(".policy-form__wrapper").within(() => {
        cy.findByRole("button", { name: /run/i }).should("not.exist");
        cy.findByRole("button", { name: /save/i }).should("not.exist");
      });
    });
  });

  describe("Settings tests", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", "user123#");
      cy.visit("/settings/users");
    });

    it("Can verify user does not have access to settings", () => {
      cy.visit("/settings/organization");
      cy.findByText(/you do not have permissions/i).should("exist");
    });
  });

  describe("Profile tests", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", "user123#");
      cy.visit("/profile");
    });

    it("Can verify teams is disabled for the Profile page", () => {
      cy.getAttached(".user-settings__additional").within(() => {
        cy.findByText(/teams/i).should("not.exist");
      });
    });

    it("Can verify the role of the user is observer", () => {
      cy.getAttached(".user-settings__additional").within(() => {
        cy.findByText("Role")
          .next()
          .contains(/observer/i);
      });
    });

    // nav restrictions are at the end because we expect to see a
    // 403 error overlay which will hide the nav and make the test fail
    cy.visit("/dashboard");
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

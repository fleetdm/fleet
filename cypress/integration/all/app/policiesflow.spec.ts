describe("Policies flow (empty)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("Manage policies page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/policies/manage");
    });
    it("creates a custom policy", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.findByText(/create your own policy/i).click();
      cy.getAttached(".ace_scroller")
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM users WHERE username = 'backup' LIMIT 1;"
        );
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.getAttached(".policy-form__policy-save-modal-name")
        .click()
        .type("Does the device have a user named 'backup'?");
      cy.getAttached(".policy-form__policy-save-modal-description")
        .click()
        .type("Returns yes or no for having a user named 'backup'");
      cy.getAttached(".policy-form__policy-save-modal-resolution")
        .click()
        .type("Create a user named 'backup'");
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.findByText(/policy created/i).should("exist");
    });
    it("creates a default policy", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.findByText(/gatekeeper enabled/i).click();
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.findByRole("button", { name: /^Save$/ }).click();

      cy.findByText(/policy created/i).should("exist");
    });
  });
});

describe("Policies flow (seeded)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedPolicies();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("Manage policies page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/policies/manage");
    });
    it("links to manage host page filtered by policy", () => {
      cy.getAttached(".failing_host_count__cell")
        .first()
        .within(() => {
          cy.getAttached(".button--text-link").click();
        });
      // confirm policy functionality on manage host page
      cy.getAttached(".manage-hosts__policies-filter-block").within(() => {
        cy.findByText(/filevault enabled/i).should("exist");
        cy.findByText(/no/i).should("exist").click();
        cy.findByText(/yes/i).should("exist");
        cy.get('img[alt="Remove policy filter"]').click();
        cy.findByText(/filevault enabled'/i).should("not.exist");
      });
    });
    it("edits an existing policy", () => {
      cy.getAttached(".name__cell .button--text-link").last().click();
      cy.getAttached(".ace_scroller")
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;"
        );
      cy.getAttached(".policy-form__save").click();
      cy.findByText(/policy updated/i).should("exist");
    });

    it("deletes an existing policy", () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.getAttached(".fleet-checkbox__input").check({ force: true });
          });
      });
      cy.findByRole("button", { name: /delete/i }).click();
      cy.getAttached(".remove-policies-modal").within(() => {
        cy.findByRole("button", { name: /cancel/i }).should("exist");
        cy.findByRole("button", { name: /delete/i }).click();
      });
      cy.findByText(/removed policy/i).should("exist");
      cy.findByText(/backup/i).should("not.exist");
    });
    it("creates a failing policies webhook", () => {
      cy.findByRole("button", { name: /manage automations/i }).click();
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-checkbox__input").check({ force: true });
      });
      cy.getAttached("#webhook-url").click().type("www.foo.com/bar");
      cy.findByRole("button", { name: /^Save$/ }).click();
      // Confirm failing policies webhook was added successfully
      cy.findByText(/updated policy automations/i).should("exist");
      cy.findByRole("button", { name: /manage automations/i }).click();
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-checkbox__input").should("be.checked");
      });
    });
  });
});

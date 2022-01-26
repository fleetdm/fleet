describe("Teams flow (empty)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });
  describe("Teams settings page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/settings/teams");
    });
    it("creates a new team", () => {
      cy.getAttached(".no-teams__create-button").click();
      cy.findByLabelText(/team name/i)
        .click()
        .type("Valor");
      cy.getAttached(".create-team-modal__btn-wrap").within(() => {
        // ^$ forces exact match
        cy.findByRole("button", { name: /^create$/i }).click();
      });
      cy.findByText(/successfully created valor/i).should("exist");
    });
  });
});

describe("Teams flow (seeded)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedPremium();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });
  describe("Teams settings page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/settings/teams");
    });
    it("edit name of an existing team", () => {
      cy.getAttached(".table-container").within(() => {
        cy.contains("Apples");
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.getAttached(".Select-arrow-zone").click();
              cy.findByText(/edit/i).click({ force: true });
            });
        });
      });
      cy.getAttached(".edit-team-modal").within(() => {
        cy.findByLabelText(/team name/i)
          .clear()
          .type("Bananas");
        cy.findByRole("button", { name: /save/i }).click();
      });
      cy.findByText(/successfully edited/i).should("be.visible");
      cy.findByText(/apples/i).should("not.exist");
    });
    it("deletes an existing team", () => {
      cy.getAttached(".table-container").within(() => {
        cy.contains("Bananas");
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.getAttached(".Select-arrow-zone").click();
              cy.findByText(/delete/i).click({ force: true });
            });
        });
      });
      cy.getAttached(".delete-team-modal__btn-wrap").within(() => {
        cy.findByRole("button", { name: /delete/i }).click();
      });
      cy.findByText(/successfully deleted/i).should("be.visible");
      cy.findByText(/bananas/i).should("not.exist");
    });
  });
  // describe("Manage schedules page", () => {
  //   beforeEach(() => {
  //     cy.loginWithCySession();
  //     cy.visit("/schedule/manage");
  //   });
  //   it("adds a query to team schedule", () => {
  //     cy.getAttached(".no-schedule__schedule-button").click();
  //     // TODO: Unable to add tests because "Schedule a query" button detattaches even when using `getAttached`
  //   });
  // });
  describe("Team details page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/settings/teams");
      cy.getAttached(".table-container").within(() => {
        cy.contains("Oranges").click({ force: true });
      });
    });
    it("allows to add new enroll secret to team", () => {
      cy.getAttached(".team-details__team-actions")
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
    it("allows to see and click generate installer", () => {
      cy.getAttached(".team-details__team-actions")
        .contains("button", /generate installer/i)
        .click();
      cy.getAttached(".modal__content").contains("button", /done/i).click();
    });
    it("edit agent options an existing team", () => {
      cy.findByText(/agent options/i).click();
      cy.contains(".ace_content", "config:");
      cy.get(".ace_text-input")
        .first()
        .focus()
        .type("{selectall}{backspace}config:\n  options:");

      cy.findByRole("button", { name: /save options/i }).click();

      cy.contains("span", /successfully saved/i).should("exist");
      cy.visit("/settings/teams/2/options");

      cy.contains(/config:/i).should("be.visible");
      cy.contains(/options:/i).should("be.visible");
    });
  });
});

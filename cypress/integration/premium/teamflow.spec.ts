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
      cy.getAttached(".create-team-modal .modal-cta-wrap").within(() => {
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
    cy.seedQueries();
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
    it("edits name of an existing team", () => {
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
      cy.findByText(/updated team name/i).should("be.visible");
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
      cy.getAttached(".delete-team-modal .modal-cta-wrap").within(() => {
        cy.findByRole("button", { name: /delete/i }).click();
      });
      cy.findByText(/successfully deleted/i).should("be.visible");
      cy.findByText(/bananas/i).should("not.exist");
    });
  });
  describe("Manage schedules page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/schedule/manage");
    });
    it("adds a query to team schedule", () => {
      cy.getAttached(".manage-schedule-page__header").within(() => {
        cy.contains("All teams").click({ force: true });
        cy.contains("Oranges").click({ force: true });
      });
      cy.getAttached(".no-schedule__schedule-button").click();
      cy.getAttached(".schedule-editor-modal__form").within(() => {
        cy.findByText(/select query/i).click();
        cy.findByText(/detect presence/i).click();
        cy.findByText(/every day/i).click();
        cy.findByText(/every 6 hours/i).click();
        cy.findByText(/show advanced options/i).click();
        cy.findByText(/snapshot/i).click();
        cy.findByText(/ignore removals/i).click();
        cy.getAttached(".schedule-editor-modal__form-field--platform").within(
          () => {
            cy.findByText(/all/i).click();
            cy.findByText(/linux/i).click();
          }
        );
        cy.getAttached(
          ".schedule-editor-modal__form-field--osquer-vers"
        ).within(() => {
          cy.findByText(/all/i).click();
          cy.findByText(/4.6.0/i).click();
        });
        cy.getAttached(".schedule-editor-modal__form-field--shard").within(
          () => {
            cy.getAttached(".input-field").click().type("50");
          }
        );
        cy.getAttached(".modal-cta-wrap").within(() => {
          cy.findByRole("button", { name: /schedule/i }).click();
        });
      });
      cy.findByText(/successfully added/i).should("be.visible");
      cy.getAttached("tbody>tr").should("have.length", 1);
    });
  });
  describe("Team details page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/settings/teams");
      cy.getAttached(".table-container").within(() => {
        cy.contains("Oranges").click({ force: true });
      });
    });
    it("allows to add new enroll secret to team", () => {
      cy.getAttached(".team-details__action-buttons--secondary-buttons")
        .contains("button", /manage enroll secret/i)
        .click();
      cy.getAttached(".enroll-secret-modal__add-secret")
        .contains("button", /add secret/i)
        .click();
      cy.getAttached(".secret-editor-modal .modal-cta-wrap")
        .contains("button", /save/i)
        .click();
      cy.getAttached(".enroll-secret-modal .modal-cta-wrap")
        .contains("button", /done/i)
        .click();
    });
    it("allows to see and click 'Add hosts'", () => {
      cy.getAttached(".team-details__action-buttons--primary")
        .contains("button", /add hosts/i)
        .click();
      cy.getAttached(".modal__content").contains("button", /done/i).click();
    });
    it("edits agent options of an existing team", () => {
      cy.findByText(/agent options/i).click();
      cy.contains(".ace_content", "config:");
      cy.getAttached(".ace_text-input")
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

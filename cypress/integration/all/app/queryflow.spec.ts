describe("Query flow (empty)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });
  describe("Manage queries page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/queries/manage");
    });
    it("creates a new query", () => {
      cy.getAttached(".queries-list-wrapper__create-button").click();
      cy.getAttached(".ace_scroller")
        .click({ force: true })
        .type("{selectall}SELECT * FROM windows_crashes;");
      cy.findByRole("button", { name: /save/i }).click();
      cy.getAttached(".query-form__query-save-modal-name")
        .click()
        .type("Query all window crashes");
      cy.getAttached(".query-form__query-save-modal-description")
        .click()
        .type("See all window crashes");
      cy.findByRole("button", { name: /save query/i }).click();
      cy.findByText(/query created/i).should("exist");
      cy.getAttached(".query-form__query-name").within(() => {
        cy.findByText(/query all window crashes/i).should("exist");
      });
    });
  });
});

describe("Query flow (seeded)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedQueries();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });
  describe("Manage queries page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/queries/manage");
    });
    it("edits an existing query", () => {
      cy.getAttached(".name__cell .button--text-link").first().click();
      cy.findByText(/run query/i).should("exist");
      cy.getAttached(".ace_scroller")
        .click()
        .type("{selectall}SELECT datetime, username FROM windows_crashes;");
      cy.getAttached(".button--brand.query-form__save").click();
      cy.findByText(/query updated/i).should("be.visible");
    });
    it("deletes an existing query", () => {
      cy.findByText(/detect linux hosts/i)
        .parent()
        .parent()
        .within(() => {
          cy.getAttached(".fleet-checkbox__input").check({
            force: true,
          });
        });
      cy.findByRole("button", { name: /delete/i }).click();
      cy.getAttached(".button--alert.remove-query-modal__btn").click();
      cy.findByText(/successfully removed query/i).should("be.visible");
      cy.findByText(/detect linux hosts/i).should("not.exist");
    });
  });
  describe("Manage schedules page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/schedule/manage");
    });
    it("creates a new scheduled query", () => {
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
        cy.getAttached(".schedule-editor-modal__btn-wrap").within(() => {
          cy.findByRole("button", { name: /schedule/i }).click();
        });
      });
      cy.findByText(/successfully added/i).should("be.visible");
    });

    it("edit a scheduled query successfully", () => {
      cy.getAttached("tbody>tr")
        .should("have.length", 1)
        .within(() => {
          cy.findByText(/action/i).click();
          cy.findByText(/edit/i).click();
        });
      cy.getAttached(".schedule-editor-modal__form").within(() => {
        cy.findByText(/every 6 hours/i).click();
        cy.findByText(/every day/i).click();

        cy.getAttached(".schedule-editor-modal__btn-wrap").within(() => {
          cy.findByRole("button", { name: /schedule/i }).click();
        });
      });
      cy.findByText(/successfully updated/i).should("be.visible");
    });

    it("remove a scheduled query successfully", () => {
      cy.getAttached("tbody>tr")
        .should("have.length", 1)
        .within(() => {
          cy.findByText(/1 day/i).should("exist");
          cy.findByText(/action/i).click();
          cy.findByText(/remove/i).click();
        });
      cy.getAttached(".remove-scheduled-query-modal__btn-wrap").within(() => {
        cy.findByRole("button", { name: /remove/i }).click();
      });
      cy.findByText(/successfully removed/i).should("be.visible");
    });
  });
});

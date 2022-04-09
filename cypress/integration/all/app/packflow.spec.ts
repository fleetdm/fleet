describe("Pack flow (empty)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("Manage packs page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/packs/manage");
    });
    it("creates a new pack", () => {
      cy.findByRole("button", { name: /create new pack/i }).click();
      cy.findByLabelText(/name/i).click().type("Errors and crashes");
      cy.findByLabelText(/description/i)
        .click()
        .type("See all user errors and window crashes.");
      cy.findByRole("button", { name: /save query pack/i }).click();
    });
  });
});
describe("Pack flow (seeded)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedQueries();
    cy.seedPacks();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("Pack details page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/packs/manage");
      cy.getAttached(".name__cell > .button--text-link").first().click();
    });
    it("adds a query to an existing pack", () => {
      cy.findByRole("button", { name: /add query/i }).click();
      cy.findByText(/select query/i).click();
      cy.findByText(/get authorized/i).click();
      cy.getAttached(
        ".pack-query-editor-modal__form-field--frequency > .input-field"
      )
        .click()
        .type("3600");
      cy.getAttached(
        ".pack-query-editor-modal__form-field--osquer-vers > .Select"
      ).click();
      cy.findByText(/4.7/i).click();
      cy.getAttached(
        ".pack-query-editor-modal__form-field--shard > .input-field"
      )
        .click()
        .type("50");
      cy.getAttached(".pack-query-editor-modal__btn-wrap")
        .contains("button", /add query/i)
        .click();
      cy.findByText(/get authorized/i).should("exist");
    });
    it("removes a query from an existing pack", () => {
      cy.getAttached(".fleet-checkbox__input").check({ force: true });
      cy.findByRole("button", { name: /remove/i }).click();
      cy.getAttached(".remove-pack-query-modal__btn-wrap")
        .contains("button", /remove/i)
        .click();
    });
    it("edits an existing pack", () => {
      cy.findByLabelText(/name/i).clear().type("Server errors");
      cy.findByLabelText(/description/i)
        .clear()
        .type("See all server errors.");
      cy.findByRole("button", { name: /save/i }).click();
    });
  });
  describe("Manage packs page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/packs/manage");
    });
    it("deletes an existing pack", () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.getAttached(".fleet-checkbox__input").check({ force: true });
          });
      });
      cy.findByRole("button", { name: /delete/i }).click();
      cy.getAttached(".remove-pack-modal__btn-wrap > .button--alert")
        .contains("button", /delete/i)
        .click({ force: true });
      cy.findByText(/successfully deleted/i).should("be.visible");
      cy.visit("/packs/manage");
      cy.getAttached(".table-container").within(() => {
        cy.findByText(/windows starter pack/i).should("not.exist");
      });
    });
  });
});

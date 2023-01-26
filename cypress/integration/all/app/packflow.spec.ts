/* NOTE: Product decision to remove packs from UI

import managePacksPage from "../../pages/managePacksPage";

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
      managePacksPage.visitsManagePacksPage();
    });
    it("creates a new pack", () => {
      managePacksPage.allowsCreatePack();
      managePacksPage.verifiesCreatedPack();
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
      managePacksPage.visitsManagePacksPage();
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
      cy.getAttached(".pack-query-editor-modal .modal-cta-wrap")
        .contains("button", /add query/i)
        .click();
      cy.findByText(/get authorized/i).should("exist");
    });
    it("removes a query from an existing pack", () => {
      cy.getAttached(".fleet-checkbox__input").check({ force: true });
      cy.findByRole("button", { name: /remove/i }).click();
      cy.getAttached(".remove-pack-query-modal .modal-cta-wrap")
        .contains("button", /remove/i)
        .click();
    });
    it("edits an existing pack", () => {
      managePacksPage.allowsEditPack();
      managePacksPage.verifiesEditedPack();
    });
  });
  describe("Manage packs page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      managePacksPage.visitsManagePacksPage();
    });
    it("deletes an existing pack", () => {
      managePacksPage.allowsDeletePack();
      managePacksPage.verifiesDeletedPack();
    });
  });
});
*/

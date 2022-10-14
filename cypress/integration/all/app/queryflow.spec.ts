import manageQueriesPage from "../../pages/manageQueriesPage";
import manageSchedulePage from "../../pages/manageSchedulePage";

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
      manageQueriesPage.visitManageQueriesPage();
    });
    it("creates a new query", () => {
      manageQueriesPage.allowsCreateNewQuery();
      manageQueriesPage.verifiesCreatedNewQuery();
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
      manageQueriesPage.visitManageQueriesPage();
    });
    it("runs a live query and allows exporting results", () => {
      cy.addDockerHost();
      manageQueriesPage.allowsRunQuery();
      manageQueriesPage.verifiesRanQuery();
      manageQueriesPage.allowsViewRanQuery();
      manageQueriesPage.allowsExportQueryResults();
      cy.stopDockerHost();
    });
    it("edits an existing query", () => {
      manageQueriesPage.allowsEditExistingQuery();
      manageQueriesPage.verifiesEditedExistingQuery();
    });
    it("saves an existing query as new query", () => {
      manageQueriesPage.allowsSaveAsNewQuery();
      manageQueriesPage.verifiesSavedAsNewQuery();
    });
    it("deletes an existing query", () => {
      manageQueriesPage.allowsDeleteExistingQuery();
      manageQueriesPage.verifiesDeletedExistingQuery();
    });
  });
  describe("Manage schedules page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      manageSchedulePage.visitManageSchedulePage();
    });
    it("creates a new scheduled query", () => {
      manageSchedulePage.allowsAddSchedule();
      manageSchedulePage.verifiesAddedSchedule();
    });

    it("shows sql of a scheduled query successfully", () => {
      cy.getAttached("tbody>tr")
        .should("have.length", 1)
        .within(() => {
          cy.findByText(/action/i).click();
          cy.findByText(/show query/i).click();
        });
      cy.getAttached(".show-query-modal").within(() => {
        cy.getAttached(".ace_content").within(() => {
          cy.contains(/select/i).should("exist");
          cy.contains(/cypress/i).should("exist");
        });
      });
    });

    it("edit a scheduled query successfully", () => {
      manageSchedulePage.allowsEditSchedule();
      manageSchedulePage.verifiesEditedSchedule();
    });

    it("remove a scheduled query successfully", () => {
      manageSchedulePage.allowsRemoveSchedule();
      manageSchedulePage.verifiesRemovedSchedule();
    });
  });
});

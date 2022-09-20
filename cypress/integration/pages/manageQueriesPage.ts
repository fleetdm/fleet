const manageQueriesPage = {
  visitManageQueriesPage: () => {
    cy.visit("/queries/manage");
  },

  createsNewQuery: () => {
    cy.getAttached(".queries-table__create-button").click();
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
  },
};

export default manageQueriesPage;

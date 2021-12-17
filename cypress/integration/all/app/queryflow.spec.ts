describe(
  "Query flow",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
    });

    it("Create, check, edit, and delete a query successfully and create, edit, and delete a global scheduled query successfully", () => {
      cy.visit("/queries/manage");

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
      cy.findByText(/back to queries/i).should("exist");
      cy.visit("/queries/manage");
      cy.getAttached(".name__cell .button--text-link").first().click();

      cy.findByText(/run query/i).should("exist");

      cy.getAttached(".ace_scroller")
        .click()
        .type("{selectall}SELECT datetime, username FROM windows_crashes;");

      cy.getAttached(".button--brand.query-form__save").click();

      cy.findByText(/query updated/i).should("be.visible");

      cy.visit("/queries/manage");

      cy.findByText(/query all window crashes/i)
        .parent()
        .parent()
        .within(() => {
          cy.getAttached(".fleet-checkbox__input").check({ force: true });
        });

      cy.findByRole("button", { name: /delete/i }).click();

      cy.getAttached(".button--alert.remove-query-modal__btn").click();

      cy.findByText(/successfully removed query/i).should("be.visible");
      cy.getAttached(".name__cell .button--text-link").should("not.exist");
    });
  }
);

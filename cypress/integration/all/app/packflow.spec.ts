describe(
  "Pack flow",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedQueries();
    });

    it("Create, edit, and delete a pack and pack query successfully", () => {
      // Create pack
      cy.visit("/packs/manage");

      cy.findByRole("button", { name: /create new pack/i }).click();

      cy.findByLabelText(/name/i).click().type("Errors and crashes");

      cy.findByLabelText(/description/i)
        .click()
        .type("See all user errors and window crashes.");

      cy.findByRole("button", { name: /save query pack/i }).click();

      // Add query to pack
      cy.visit("/packs/manage");

      cy.getAttached(".name__cell > .button--text-link").click();

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

      // Remove query from pack
      cy.findByText(/get authorized/i).should("exist");
      cy.getAttached(".fleet-checkbox__input").check({ force: true });

      cy.findByRole("button", { name: /remove/i }).click();
      cy.getAttached(".remove-pack-query-modal__btn-wrap")
        .contains("button", /remove/i)
        .click();

      // Edit pack
      cy.visit("/packs/manage");

      cy.getAttached(".name__cell > .button--text-link").click();

      cy.findByLabelText(/name/i).clear().type("Server errors");

      cy.findByLabelText(/description/i)
        .clear()
        .type("See all server errors.");

      cy.findByRole("button", { name: /save/i }).click();

      // Delete pack
      cy.visit("/packs/manage");

      cy.getAttached(".fleet-checkbox__input").check({ force: true });

      cy.findByRole("button", { name: /delete/i }).click();

      cy.getAttached(".remove-pack-modal__btn-wrap > .button--alert")
        .contains("button", /delete/i)
        .click({ force: true });

      cy.findByText(/successfully deleted/i).should("be.visible");

      cy.visit("/packs/manage");

      cy.getAttached(".table-container").within(() => {
        cy.findByText(/server errors/i).should("not.exist");
      });
    });
  }
);

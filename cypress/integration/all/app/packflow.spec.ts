describe(
  "Pack flow",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
    });

    it("Create, edit, and delete a pack and pack query successfully", () => {
      cy.visit("/packs/manage");

      cy.findByRole("button", { name: /create new pack/i }).click();

      cy.findByLabelText(/name/i).click().type("Errors and crashes");

      cy.findByLabelText(/description/i)
        .click()
        .type("See all user errors and window crashes.");

      cy.findByRole("button", { name: /save query pack/i }).click();

      cy.visit("/packs/manage");

      cy.getAttached(".name__cell > .button--text-link").click();

      cy.findByLabelText(/name/i)
        .click()
        .type("{selectall}{backspace}Server errors");

      cy.findByLabelText(/description/i)
        .click()
        .type("{selectall}{backspace}See all server errors.");

      cy.findByRole("button", { name: /save/i }).click();

      cy.visit("/packs/manage");

      cy.get(".fleet-checkbox__input").check({ force: true });

      cy.getAttached(".selection__header .fleet-checkbox__tick").click();

      cy.visit("/queries/manage");

      cy.getAttached(".queries-list-wrapper__create-button").click();

      cy.getAttached(".ace_scroller")
        .click({ force: true })
        .type("{selectall}SELECT * FROM windows_crashes;");

      cy.findByRole("button", { name: /save/i }).click();

      cy.findByLabelText(/name/i).click().type("Query all window crashes");

      cy.findByLabelText(/description/i)
        .click()
        .type("See all window crashes");

      cy.findByRole("button", { name: /save query/i }).click();

      cy.visit("/packs/manage");

      cy.getAttached(".name__cell > .button--text-link").click();

      cy.findByRole("button", { name: /add query/i }).click();

      cy.findByText(/select query/i).click();
      cy.findByText(/query all/i).click();
      cy.getAttached(".pack-query-editor-modal__form-field--frequency > .input-field")
        .click()
        .type("3600");
      cy.getAttached(
        ".pack-query-editor-modal__form-field--osquer-vers > .Select"
      ).click();
      cy.findByText(/4.7/i).click();
      cy.getAttached(".pack-query-editor-modal__form-field--shard > .input-field")
        .click()
        .type("50");

      cy.getAttached(".pack-query-editor-modal__btn-wrap")
        .contains("button", /add query/i)
        .click();

      cy.findByText(/query all window crashes/i).should("exist");
      cy.getAttached(".fleet-checkbox__input").check({ force: true });

      cy.findByRole("button", { name: /remove/i }).click();
      cy.getAttached(".remove-pack-query-modal__btn-wrap")
        .contains("button", /remove/i)
        .click();

      cy.visit("/packs/manage");

      cy.getAttached(".fleet-checkbox__input").check({ force: true });

      cy.findByRole("button", { name: /delete/i }).click();

      cy.getAttached(".remove-pack-modal__btn-wrap > .button--alert")
        .contains("button", /delete/i)
        .click({ force: true });

      cy.findByText(/successfully deleted/i).should("be.visible");

      cy.visit("/packs/manage");

      cy.findByText(/server errors/i).should("not.exist");
    });
  }
);

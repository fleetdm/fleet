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

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByRole("button", { name: /create new pack/i }).click();

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByLabelText(/name/i).click().type("Errors and crashes");

      cy.findByLabelText(/description/i)
        .click()
        .type("See all user errors and window crashes.");

      cy.findByRole("button", { name: /save query pack/i }).click();

      cy.visit("/packs/manage");
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting

      cy.findByText(/errors and crashes/i).click();

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByLabelText(/name/i)
        .click()
        .type("{selectall}{backspace}Server errors");

      cy.findByLabelText(/description/i)
        .click()
        .type("{selectall}{backspace}See all server errors.");

      cy.findByRole("button", { name: /save/i }).click();

      cy.visit("/packs/manage");

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.get(".fleet-checkbox__input").check({ force: true });

      cy.findByRole("button", { name: /disable/i }).click();

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByText(/disabled/i).should("exist");

      cy.visit("/queries/manage");

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByRole("button", { name: /create new query/i }).click();

      // Using class selector because third party element doesn't work with Cypress Testing Selector Library
      cy.get(".ace_scroller")
        .click({ force: true })
        .type("{selectall}SELECT * FROM windows_crashes;");

      cy.findByRole("button", { name: /save/i }).click();

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByLabelText(/name/i).click().type("Query all window crashes");

      cy.findByLabelText(/description/i)
        .click()
        .type("See all window crashes");

      cy.findByRole("button", { name: /save query/i }).click();

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.visit("/packs/manage");

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByText(/server errors/i).click();

      cy.findByRole("button", { name: /add query/i }).click();

      cy.findByText(/select query/i).click();
      cy.findByText(/query all/i).click();
      cy.get(".pack-query-editor-modal__form-field--frequency > .input-field")
        .click()
        .type("3600");
      cy.get(
        ".pack-query-editor-modal__form-field--osquer-vers > .Select"
      ).click();
      cy.findByText(/4.7/i).click();
      cy.get(".pack-query-editor-modal__form-field--shard > .input-field")
        .click()
        .type("50");
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting

      cy.get(".pack-query-editor-modal__btn-wrap")
        .contains("button", /add query/i)
        .click();

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByText(/query all window crashes/i).should("exist");
      cy.get(".fleet-checkbox__input").check({ force: true });

      cy.findByRole("button", { name: /remove/i }).click();
      cy.get(".remove-pack-query-modal__btn-wrap")
        .contains("button", /remove/i)
        .click();

      cy.visit("/packs/manage");

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.get(".fleet-checkbox__input").check({ force: true });

      cy.findByRole("button", { name: /delete/i }).click();

      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting

      cy.get(".remove-pack-modal__btn-wrap > .button--alert")
        .contains("button", /delete/i)
        .click({ force: true });

      cy.findByText(/successfully deleted/i).should("be.visible");

      cy.findByText(/server errors/i).should("not.exist");
    });
  }
);

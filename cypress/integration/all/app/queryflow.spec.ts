describe("Query flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
  });

  it("Create, check, edit, and delete a query successfully", () => {
    cy.visit("/queries/manage");

    cy.findByRole("button", { name: /create new query/i }).click();

    cy.findByLabelText(/query name/i)
      .click()
      .type("Query all window crashes");

    // Using class selector because third party element doesn't work with Cypress Testing Selector Library
    cy.get(".ace_scroller")
      .click({ force: true })
      .type("{selectall}{backspace}SELECT * FROM windows_crashes;");

    cy.findByLabelText(/description/i)
      .click()
      .type("See all window crashes");

    cy.findByRole("button", { name: /save/i }).click();

    cy.findByRole("button", { name: /save as new/i }).click();

    // Just refreshes to create new query, needs success alert to user that they created a query

    cy.visit("/queries/manage");

    cy.findByText(/query all/i).click();

    cy.findByRole("button", { name: /edit or run query/i }).click();

    cy.get(".ace_scroller")
      .click({ force: true })
      .type(
        "{selectall}{backspace}SELECT datetime, username FROM windows_crashes;"
      );

    cy.findByRole("button", { name: /save/i }).click();

    cy.findByRole("button", { name: /save changes/i }).click();

    cy.findByText(/query updated/i).should("be.visible");

    // Test Schedules
    cy.visit("/schedule/manage");

    cy.wait(1000);

    cy.findByRole("button", { name: /schedule a query/i }).click();

    cy.findByText(/select query/i).click();

    cy.findByText(/query all window crashes/i).click();

    cy.get(
      ".schedule-editor-modal__form-field--frequency > .dropdown__select"
    ).click();

    cy.findByText(/every week/i).click();

    cy.findByText(/show advanced options/i).click();

    cy.get(
      ".schedule-editor-modal__form-field--logging > .dropdown__select"
    ).click();

    cy.findByText(/ignore removals/i).click();

    cy.get(".schedule-editor-modal__form-field--shard > .input-field")
      .click()
      .type("50");

    cy.get(".schedule-editor-modal__btn-wrap")
      .contains("button", /schedule/i)
      .click();

    cy.visit("/schedule/manage");

    cy.findByText(/query all window crashes/i).should("exist");

    // Checkbox won't check so can't test remove schedule
    // cy.get("tbody").get(".table-checkbox__input").click();

    // cy.findByRole("button", { name: /remove query/i }).click();

    // cy.get(".remove-scheduled-query-modal__btn-wrap")
    //   .contains("button", /remove/i)
    //   .click();

    // cy.findByText(/query all window crashes/i).should("not.exist");
    // End Test Schedules

    cy.visit("/queries/manage");

    // This element has no label, text, or role
    cy.get("#query-checkbox-1").check({ force: true });

    cy.findByRole("button", { name: /delete/i }).click();

    // Can't figure out how attach findByRole onto modal button
    // Can't use findByText because delete button under modal
    cy.get(".manage-queries-page__modal-btn-wrap > .button--alert")
      .contains("button", /delete/i)
      .click();

    cy.findByText(/successfully deleted/i).should("be.visible");

    cy.findByText(/query all/i).should("not.exist");
  });
});

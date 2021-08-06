describe("Query flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
  });

  it("Create, check, edit, and delete a query successfully and create, edit, and delete a global scheduled query successfully", () => {
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

    cy.findByText(/edit & run query/i).should("exist");

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

    cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting

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

    cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
    cy.findByText(/actions/i).click();
    cy.findByText(/edit/i).click();

    cy.get(
      ".schedule-editor-modal__form-field--frequency > .dropdown__select"
    ).click();

    cy.findByText(/every 6 hours/i).click();

    cy.findByText(/show advanced options/i).click();

    cy.findByText(/ignore removals/i).click();
    cy.findByText(/snapshot/i).click();

    cy.get(".schedule-editor-modal__form-field--shard > .input-field")
      .click()
      .type("{selectall}{backspace}10");

    cy.get(".schedule-editor-modal__btn-wrap")
      .contains("button", /schedule/i)
      .click();

    cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
    cy.findByText(/actions/i).click();
    cy.findByText(/remove/i).click();

    cy.get(".remove-scheduled-query-modal__btn-wrap")
      .contains("button", /remove/i)
      .click();

    cy.findByText(/query all window crashes/i).should("not.exist");
    // End Test Schedules

    cy.visit("/queries/manage");

    cy.findByText(/query all window crashes/i)
      .parent()
      .parent()
      .within(() => {
        cy.get(".fleet-checkbox__input").check({ force: true });
      });

    cy.findByRole("button", { name: /delete/i }).click();

    // Can't figure out how attach findByRole onto modal button
    // Can't use findByText because delete button under modal
    cy.get(".remove-query-modal")
      .contains("button", /delete/i)
      .click();

    cy.findByText(/successfully removed query/i).should("be.visible");

    cy.findByText(/query all/i).should("not.exist");
  });
});

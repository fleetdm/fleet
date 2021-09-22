describe("Teams flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.viewport(1200, 660);
  });

  /* TODO fix and reenable
  This test is causing major flake issues due to the dropdown menu

  it("Create, edit, and delete a team successfully", () => {
    cy.visit("/settings/teams");

    cy.findByRole("button", { name: /create team/i }).click({ force: true });

    cy.findByLabelText(/team name/i)
      .click()
      .type("Valor");

    // ^$ forces exact match
    cy.findByRole("button", { name: /^create$/i }).click();

    cy.visit("/settings/teams");
    // Allow rendering to settle
    // TODO this might represent a bug in the React code.
    cy.wait(100); // eslint-disable-line cypress/no-unnecessary-waiting

    cy.contains("Valor").click({ force: true });

    cy.findByText(/agent options/i).click();

    cy.contains(".ace_content", "config:");
    cy.get(".ace_text-input")
      .first()
      .focus()
      .type("{selectall}{backspace}config:\n  options:");

    cy.findByRole("button", { name: /save options/i }).click();

    cy.contains("span", /successfully saved/i);

    cy.visit("/settings/teams/1/options");

    cy.contains(/config:/i).should("be.visible");
    cy.contains(/options:/i).should("be.visible");

    // Check team in schedules
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

    cy.wait(2000); // eslint-disable-line cypress/no-unnecessary-waiting
    cy.findByText(/all teams/i).click();
    cy.findByText(/valor/i).click();

    cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
    cy.findByText(/query all window crashes/i).should("not.exist");
    cy.findByText(/inherited query/i).click();
    cy.findByText(/query all window crashes/i).should("exist");

    // Edit Team
    cy.visit("/settings/teams");

    cy.contains("Valor").get(".Select-arrow-zone").click();

    // need force:true for dropdown
    cy.findByText(/edit/i).click({ force: true });

    cy.findByLabelText(/team name/i)
      .click()
      .type("{selectall}{backspace}Mystic");

    cy.findByRole("button", { name: /save/i }).click();

    cy.visit("/settings/teams");
    // Allow rendering to settle
    // TODO this might represent a bug in the React code.
    cy.wait(100); // eslint-disable-line cypress/no-unnecessary-waiting

    cy.contains("Mystic").get(".Select-arrow-zone").click();

    cy.findByText(/delete/i).click({ force: true });

    cy.findByRole("button", { name: /delete/i }).click();

    cy.findByText(/successfully deleted/i).should("be.visible");

    cy.visit("/settings/teams");
    // Allow rendering to settle
    // TODO this might represent a bug in the React code.
    cy.wait(100); // eslint-disable-line cypress/no-unnecessary-waiting

    cy.findByText(/mystic/i).should("not.exist");
  });
  */
});

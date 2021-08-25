describe("Teams flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.viewport(1200, 660);
  });

  it("Create, edit, and delete a team successfully", () => {
    cy.visit("/settings/teams");

    cy.findByRole("button", { name: /create team/i }).click();

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

    cy.get(".ace_content")
      .click()
      .type("{selectall}{backspace}apiVersion: v1{enter}kind: options");

    cy.findByRole("button", { name: /save/i }).click();

    cy.visit("/settings/teams/1/options");

    cy.contains(/apiVersion: v1/i).should("be.visible");
    cy.contains(/kind: options/i).should("be.visible");

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
});

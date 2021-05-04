describe("Teams flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
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
    cy.wait(100); // eslint-disable-line cypress/no-unnecessary-waiting

    cy.contains("Valor").get(".Select-arrow-zone").click();

    // need force:true for dropdown
    cy.findByText(/edit/i).click({ force: true });

    cy.findByLabelText(/team name/i)
      .click()
      .type("{selectall}{backspace}Mystic");

    cy.findByRole("button", { name: /save/i }).click();

    cy.visit("/settings/teams");
    // Allow rendering to settle
    cy.wait(100); // eslint-disable-line cypress/no-unnecessary-waiting

    cy.contains("Mystic").get(".Select-arrow-zone").click();

    cy.findByText(/delete/i).click({ force: true });

    cy.findByRole("button", { name: /delete/i }).click();

    cy.findByText(/successfully deleted/i).should("be.visible");

    cy.findByText(/mystic/i).should("not.exist");
  });
});

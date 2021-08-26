describe("Pack flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
  });

  it("Create, edit, and delete a pack successfully", () => {
    cy.visit("/packs/manage");

    cy.findByRole("button", { name: /create new pack/i }).click();

    cy.findByLabelText(/query pack title/i)
      .click()
      .type("Errors and crashes");

    cy.findByLabelText(/description/i)
      .click()
      .type("See all user errors and window crashes.");

    cy.findByRole("button", { name: /save query pack/i }).click();

    cy.visit("/packs/manage");

    cy.findByText(/errors and crashes/i).click();

    cy.findByText(/edit pack/i).click();

    cy.findByLabelText(/query pack title/i)
      .click()
      .type("{selectall}{backspace}Server errors");

    cy.findByLabelText(/description/i)
      .click()
      .type("{selectall}{backspace}See all server errors.");

    cy.findByRole("button", { name: /save/i }).click();

    cy.visit("/packs/manage");

    cy.get(".fleet-checkbox__input").check({ force: true });

    cy.findByRole("button", { name: /delete/i }).click();

    // Can't figure out how attach findByRole onto modal button
    // Can't use findByText because delete button under modal
    cy.get(".remove-pack-modal__btn-wrap > .button--alert")
      .contains("button", /delete/i)
      .click();

    cy.findByText(/successfully deleted/i).should("be.visible");

    cy.findByText(/server errors/i).should("not.exist");
  });
});

describe("Pack flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
  });

  it("Create, edit, and delete a pack successfully", () => {
    cy.visit("/schedule/manage");

    cy.findByRole("button", { name: /schedule a query/i }).click();

    // TODO: Schedule flow

    // specify div with class attribute since more than one schedule button
    cy.findByRole("button", { name: /schedule/i }).click();

    cy.visit("/schedule/manage");

    // TODO: Remove schedule
    // cy.get("#select-schedule-1").check({ force: true });

    // cy.findByRole("button", { name: /remove/i }).click();

    // cy.get(".all-packs-page__modal-btn-wrap > .button--alert")
    //   .contains("button", /remove/i)
    //   .click();

    // cy.findByText(/successfully removed/i).should("be.visible");

    // cy.findByText(/insert query name/i).should("not.exist");
  });
});

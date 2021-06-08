import * as path from "path";

describe("Hosts page", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
  });

  it("Add new host", () => {
    cy.visit("/");

    cy.contains("button", /add new host/i).click();

    cy.contains("a", /download/i)
      .first()
      .click();

    cy.get('a[href*="showSecret"]').click();

    // Assert enroll secret downloaded matches the one displayed
    cy.readFile(path.join(Cypress.config("downloadsFolder"), "secret.txt"), {
      timeout: 3000,
    }).then((contents) => {
      cy.get("input[disabled]").should("have.value", contents);
    });
  });
});

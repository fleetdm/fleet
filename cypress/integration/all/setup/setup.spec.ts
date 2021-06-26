describe("Setup", () => {
  // Different than normal beforeEach because we don't run the fleetctl setup.
  beforeEach(() => {
    cy.exec("make e2e-reset-db", { timeout: 5000 });
  });

  it("Completes setup", () => {
    cy.visit("/");
    cy.url().should("match", /\/setup$/);
    cy.contains(/setup/i);

    // Page 1
    cy.findByPlaceholderText(/full name/i).type("Test name");

    cy.findByPlaceholderText(/email/i).type("test@example.com");

    cy.findByPlaceholderText(/^password/i)
      .first()
      .type("admin123#");

    cy.findByPlaceholderText(/confirm password/i)
      .last()
      .type("admin123#");

    cy.contains("button:enabled", /next/i).click();

    // Page 2
    cy.findByPlaceholderText(/organization name/i).type("Fleet Test");

    cy.contains("button:enabled", /next/i).click();

    // Page 3
    cy.contains("button:enabled", /submit/i).click();

    // Page 4
    cy.contains("button:enabled", /finish/i).click();

    cy.url().should("match", /\/hosts\/manage$/i);
    cy.contains(/all hosts/i);
  });
});

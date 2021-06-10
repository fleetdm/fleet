if (Cypress.env("FLEET_TIER") === "core") {
  describe("Core tier - Admin user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedCore();
      cy.logout();
    });

    it("Can perform the appropriate actions", () => {
      cy.login("anna@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // "team" should not appear anywhere
      cy.contains(/team/i).should("not.exist");

      // See and select "add new host"
      cy.findByRole("button", { name: /new host/i }).click();
      cy.contains(/team/i).should("not.exist");
      cy.findByRole("button", { name: /done/i }).click();

      // See and select "add new label"
      cy.findByRole("button", { name: /new label/i }).click();
      cy.findByRole("button", { name: /cancel/i }).click();

      // Query page
      cy.contains("a", "Queries").click();
      cy.contains(/observers can run/i);
      cy.findByRole("button", { name: /new query/i }).click();

      // New query
      cy.findByLabelText(/query name/i)
        .click()
        .type("time");
      // ACE editor requires special handling to get typing to work sometimes
      cy.get(".ace_text-input")
        .first()
        .click({ force: true })
        .type("{selectall}{backspace}SELECT * FROM time;", { force: true });
      cy.findByLabelText(/description/i)
        .click()
        .type("Get the time.");
      cy.findByLabelText(/observers can run/i).click({ force: true });
      cy.findByRole("button", { name: /save/i }).click();
      cy.findByRole("button", { name: /new/i }).click();
      cy.contains("a", /back to queries/i).click({ force: true });
    });
  });
}

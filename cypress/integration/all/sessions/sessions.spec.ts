describe("Sessions", () => {
  // Typically we want to use a beforeEach but not much happens in these tests
  // so sharing some state should be okay and saves a bit of runtime.
  before(() => {
    cy.setup();
  });

  it("Logs in and out successfully", () => {
    cy.visit("/");
    cy.contains(/forgot password/i);

    // Log in
    cy.get("input").first().type("admin@example.com");
    cy.get("input").last().type("user123#");
    cy.get("button").click();

    // Verify dashboard
    cy.url().should("include", "/hosts/manage");
    cy.contains("All Hosts");

    // Log out
    cy.findByAltText(/user avatar/i).click();
    cy.contains("button", "Sign out").click();

    cy.url().should("match", /\/login$/);
  });

  it("Fails login with invalid password", () => {
    cy.visit("/");
    cy.get("input").first().type("admin@example.com");
    cy.get("input").last().type("bad_password");
    cy.get(".button").click();

    cy.url().should("match", /\/login$/);
    cy.contains("Authentication failed");
  });

  it("Fails to access authenticated resource", () => {
    cy.visit("/hosts/manage");

    cy.url().should("match", /\/login$/);
  });
});

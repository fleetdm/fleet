describe("Sessions", () => {
  // Typically we want to use a beforeEach but not much happens in these tests
  // so sharing some state should be okay and saves a bit of runtime.
  before(() => {
    cy.setup();
  });

  it("Logs in and out successfully", () => {
    cy.visit("/");

    cy.getAttached(".login-form__forgot-link").should("exist");

    // Log in
    cy.getAttached("input").first().type("admin@example.com");
    cy.getAttached("input").last().type("user123#");
    cy.getAttached("button").click();

    // Verify dashboard
    cy.url().should("include", "/dashboard");
    cy.contains("Host");

    // Log out
    cy.getAttached(".avatar").first().click();
    cy.contains("button", "Sign out").click();

    cy.url().should("match", /\/login$/);
  });

  it("Fails login with invalid password", () => {
    cy.visit("/");
    cy.getAttached("input").first().type("admin@example.com");
    cy.getAttached("input").last().type("bad_password");
    cy.getAttached(".button").click();

    cy.url().should("match", /\/login$/);
    cy.contains("Authentication failed");
  });

  it("Fails to access authenticated resource", () => {
    cy.visit("/hosts/manage");

    cy.url().should("match", /\/login$/);
  });
});

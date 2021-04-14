describe("SSO Sessions", () => {
  beforeEach(() => {
    cy.setup();
  });

  it("Can login with username/password", () => {
    cy.login();
    cy.setupSSO((enable_idp_login = true));
    cy.logout();

    cy.visit("/");
    cy.contains(/forgot password/i);

    // Log in
    cy.get("input").first().type("test@fleetdm.com");
    cy.get("input").last().type("admin123#");
    cy.contains("button", "Login").click();

    // Verify dashboard
    cy.url().should("include", "/hosts/manage");
    cy.contains("All Hosts");

    // Log out
    cy.findByAltText(/user avatar/i).click();
    cy.contains("button", "Sign out").click();

    cy.url().should("match", /\/login$/);
  });

  it("Can login via SSO", () => {
    cy.login();
    cy.setupSSO((enable_idp_login = true));
    cy.logout();

    cy.visit("/");

    // Log in
    cy.contains("button", "Sign On With SimpleSAML");

    cy.loginSSO();

    cy.contains("All hosts");
  });

  it("Fails when IdP login disabled", () => {
    cy.login();
    cy.setupSSO();
    cy.logout();

    cy.visit("/");

    cy.contains("button", "Sign On With SimpleSAML");

    cy.loginSSO();

    // Log in should fail
    cy.contains("Password");
  });
});

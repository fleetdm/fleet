describe("Reset user sessions flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.setupSMTP();
  });

  it("Resets a user's API tokens", () => {
    // Visit user's profile page and get current API token
    cy.visit("/profile");

    cy.getAttached(".user-settings__additional").within(() => {
      cy.findByRole("button", { name: /get api token/i }).click();
    });
    cy.getAttached(".user-settings__secret-label").within(() => {
      cy.findByText(/reveal token/i).click();
    });
    cy.getAttached(".user-settings__secret-input").within(() => {
      cy.getAttached("input").invoke("val").as("token1");
    });

    // Reset user sessions via the admin user management page
    cy.visit("/settings/users");
    // First select the table cell with the user's email address then go back up
    // to the containing row so we can select reset sessions from actions dropdown
    cy.getAttached("div.Select-placeholder", /actions/i)
      .eq(0)
      .click();
    cy.contains(/reset sessions/i).click();

    cy.get(".modal__modal_container").within(() => {
      cy.findByText(/reset sessions/i).should("exist");
      cy.findByRole("button", { name: /confirm/i }).click();
    });
    cy.findByText(/reset sessions/i).should("not.exist");

    // User should be logged out now so log in again and go to profile to get new API token
    cy.getAttached(".login-form__container").within(() => {
      cy.findByRole("button", { name: /login/i }).should("exist");
    });

    cy.login();

    cy.visit("/profile");

    cy.getAttached(".user-settings__additional").within(() => {
      cy.findByRole("button", { name: /get api token/i }).click();
    });
    cy.getAttached(".modal__content").within(() => {
      cy.findByText(/reveal token/i).click();
    });
    cy.getAttached(".user-settings__secret-input").within(() => {
      cy.get("input").invoke("val").as("token2");
    });

    // new token should not equal old token
    cy.get("@token1").then((val1) => {
      cy.get("@token2").then((val2) => {
        expect(val1).to.not.eq(val2);
      });
    });
  });
});

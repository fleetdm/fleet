describe("Reset sessions", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.setupSMTP();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("User settings page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
    });
    it("resets a user's session generating a new api token", () => {
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
      cy.visit("/settings/users");

      cy.getAttached("div.Select-placeholder", /actions/i)
        .eq(0)
        .click();
      cy.contains(/reset sessions/i).click();

      cy.get(".modal__modal_container").within(() => {
        cy.findByText(/reset sessions/i).should("exist");
        cy.findByRole("button", { name: /confirm/i }).click();
      });
      cy.findByText(/reset sessions/i).should("not.exist");

      // user should be logged out so log in for new API token
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
});

describe("Reset user sessions flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.setupSMTP();
  });

  it("Resets a user's API tokens", () => {
    // visit user's profile page and get current API token
    cy.visit("/profile");

    cy.findByRole("button", { name: /get api token/i }).click();
    cy.findByText(/reveal token/i).click();
    cy.get(".user-settings__secret-input").within(() => {
      cy.get("input").invoke("val").as("token1");
    });

    // reset user sessions via the admin user management page
    cy.visit("/settings/users");

    // first select the table cell with the user's email address then go back up to the containing row
    // so we can select reset sessions from actions dropdown
    cy.get("tbody>tr>td")
      .contains(/admin@example.com/i)
      .parent()
      .parent()
      .within(() => {
        cy.findByText(/actions/i).click();
        cy.findByText(/reset sessions/i).click();
      });

    cy.get(".modal__modal_container").within(() => {
      cy.findByText(/reset sessions/i).should("exist");
      cy.findByRole("button", { name: /confirm/i }).click();
    });
    cy.findByText(/reset sessions/i).should("not.exist");

    // user should be logged out now so log in again and go to profile to get new API token
    cy.visit("/");
    cy.findByRole("button", { name: /login/i }).should("exist");
    cy.login();

    cy.visit("/profile");
    cy.findByRole("button", { name: /get api token/i }).click();
    cy.findByText(/reveal token/i).click();
    cy.get(".user-settings__secret-input").within(() => {
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

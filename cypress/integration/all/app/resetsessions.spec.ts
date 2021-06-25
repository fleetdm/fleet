describe("Reset user sessions", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.setupSMTP();
  });

  it("Resets a user's sessions", () => {
    cy.visit("/profile");

    cy.findByRole("button", { name: /get api token/i }).click();
    cy.findByText(/reveal token/i).click();
    cy.get(".user-settings__secret-input").within(() => {
      cy.get("input").invoke("val").as("token1");
    });

    cy.visit("/settings/users");

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
    cy.findByText(/sessions reset/i).should("exist");

    cy.visit("/");
    cy.login();

    cy.visit("/profile");
    cy.findByRole("button", { name: /get api token/i }).click();
    cy.findByText(/reveal token/i).click();
    cy.get(".user-settings__secret-input").within(() => {
      cy.get("input").invoke("val").as("token2");
    });

    cy.get("@token1").then((val1) => {
      cy.get("@token2").then((val2) => {
        expect(val1).to.not.eq(val2);
      });
    });
  });
});

import CONSTANTS from "../,,/../../../support/constants";

const {
  GOOD_PASSWORD,
  BAD_PASSWORD_LENGTH,
  BAD_PASSWORD_NO_NUMBER,
  BAD_PASSWORD_NO_SYMBOL,
} = CONSTANTS;

describe("Manage users flow", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
    cy.setupSMTP();
  });
  after(() => {
    cy.logout();
  });
  describe("User management page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/settings/users");
    });
    it("searches for an existing user", () => {
      cy.findByText("admin@example.com").should("exist");
      cy.findByText("maintainer@example.com").should("exist");
      cy.findByText("observer@example.com").should("exist");
      cy.findByText("sso_user@example.com").should("exist");

      cy.findByPlaceholderText("Search").type("admin");

      cy.getAttached("tbody>tr").should("have.length", 1);
      cy.findByText("admin@example.com").should("exist");
      cy.findByText("maintainer@example.com").should("not.exist");
      cy.findByText("observer@example.com").should("not.exist");
      cy.findByText("sso_user@example.com").should("not.exist");
    });
    it("creates a new user", () => {
      cy.contains("button:enabled", /create user/i).click();
      cy.findByPlaceholderText("Full name").type("New Name");
      cy.findByPlaceholderText("Email").type("new-user@example.com");
      cy.getAttached(
        ".create-user-form__form-field--global-role > .Select"
      ).click();
      cy.getAttached(".create-user-form__form-field--global-role").within(
        () => {
          cy.findByText(/maintainer/i).click();
        }
      );
      cy.findByPlaceholderText("Password").clear().type(BAD_PASSWORD_LENGTH);
      cy.getAttached(".modal-cta-wrap")
        .contains("button", /create/i)
        .click();
      cy.findByText(/password must meet the criteria below/i).should("exist");
      cy.findByLabelText(/password must meet the criteria below/i)
        .clear()
        .type(BAD_PASSWORD_NO_NUMBER);
      cy.getAttached(".modal-cta-wrap")
        .contains("button", /create/i)
        .click();
      cy.findByText(/password must meet the criteria below/i).should("exist");
      cy.findByLabelText(/password must meet the criteria below/i)
        .clear()
        .type(BAD_PASSWORD_NO_NUMBER);
      cy.getAttached(".modal-cta-wrap")
        .contains("button", /create/i)
        .click();
      cy.findByText(/password must meet the criteria below/i).should("exist");
      cy.findByLabelText(/password must meet the criteria below/i)
        .clear()
        .type(GOOD_PASSWORD);
      cy.getAttached(".modal-cta-wrap")
        .contains("button", /create/i)
        .click();
      cy.findByText(/new name/i).should("exist");
    });
    it("edits an existing user", () => {
      cy.getAttached("tbody>tr")
        .should("have.length", 5)
        .eq(1)
        .within(() => {
          cy.findByText(/action/i).click();
          cy.findByText(/edit/i).click();
        });
      cy.findByPlaceholderText("Full name").clear().type("New Admin");
      cy.findByPlaceholderText("Email").clear().type("new-admin@example.com");
      cy.getAttached(
        ".create-user-form__form-field--global-role > .Select"
      ).click();
      cy.getAttached(".create-user-form__form-field--global-role").within(
        () => {
          cy.findByText(/admin/i).click();
        }
      );
      cy.findByLabelText("Password").clear().type(BAD_PASSWORD_LENGTH);
      cy.getAttached(".modal-cta-wrap").contains("button", /save/i).click();
      cy.findByText(/password must meet the criteria below/i).should("exist");
      cy.findByLabelText(/password must meet the criteria below/i)
        .clear()
        .type(BAD_PASSWORD_NO_NUMBER);
      cy.getAttached(".modal-cta-wrap").contains("button", /save/i).click();
      cy.findByText(/password must meet the criteria below/i).should("exist");
      cy.findByLabelText(/password must meet the criteria below/i)
        .clear()
        .type(BAD_PASSWORD_NO_SYMBOL);
      cy.getAttached(".modal-cta-wrap").contains("button", /save/i).click();
      cy.findByText(/password must meet the criteria below/i).should("exist");
      cy.findByLabelText(/password must meet the criteria below/i)
        .clear()
        .type(GOOD_PASSWORD);
      cy.getAttached(".modal-cta-wrap").contains("button", /save/i).click();
      cy.findByText(/successfully edited/i).should("exist");
    });
    it("deletes an existing user", () => {
      cy.getAttached("tbody>tr")
        .eq(1)
        .within(() => {
          cy.findByText(/new admin/i).should("exist");
          cy.findByText(/action/i).click();
          cy.findByText(/delete/i).click();
        });
      cy.getAttached(".modal-cta-wrap")
        .contains("button", /delete/i)
        .click();
      cy.findByText(/successfully deleted/i).should("exist");
      cy.getAttached("tbody>tr").should("have.length", 4);
      cy.findByText(/new-user/i).should("not.exist");
    });
  });
});

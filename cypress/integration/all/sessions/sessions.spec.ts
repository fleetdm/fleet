import CONSTANTS from "../../../support/constants";
import manageHostsPage from "../../pages/manageHostsPage";

const { GOOD_PASSWORD } = CONSTANTS;

describe("Sessions", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
  });
  it("logs in and out successfully", () => {
    cy.visit("/");
    cy.getAttached(".login-form__forgot-link").should("exist");
    // Log in
    cy.getAttached("input").first().type("admin@example.com");
    cy.getAttached("input").last().type(GOOD_PASSWORD);
    cy.getAttached("button").click();
    // Verify dashboard
    cy.url().should("include", "/dashboard");
    cy.contains("Host");
    // Log out
    cy.getAttached(".user-menu button").first().click();
    cy.contains("button", "Sign out").click();
    cy.url().should("match", /\/login$/);
  });
  it("fails login with invalid password", () => {
    cy.visit("/");
    cy.getAttached("input").first().type("admin@example.com");
    cy.getAttached("input").last().type("bad_password");
    cy.getAttached(".button").click();
    cy.url().should("match", /\/login$/);
    cy.contains("Authentication failed");
  });
  it("fails to access authenticated resource", () => {
    manageHostsPage.visitsManageHostsPage();
    cy.url().should("match", /\/login$/);
  });
});

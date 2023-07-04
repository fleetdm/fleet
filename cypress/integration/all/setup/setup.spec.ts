import CONSTANTS from "../../../support/constants";

const { GOOD_PASSWORD } = CONSTANTS;

const fillOutForm = () => {
  // Page 1
  cy.findByLabelText(/full name/i).type("Test name");

  cy.findByLabelText(/email/i).type("test@example.com");

  cy.findByLabelText(/^password/i)
    .first()
    .type(GOOD_PASSWORD);

  cy.findByLabelText(/confirm password/i)
    .last()
    .type(GOOD_PASSWORD);

  cy.contains("button:enabled", /next/i).click();

  // Page 2
  cy.findByLabelText(/organization name/i).type("Fleet Test");

  cy.contains("button:enabled", /next/i).click();

  // Page 3
  cy.contains("button:enabled", /next/i).click();

  // Page 4
  cy.contains("button:enabled", /confirm/i).click();
};

describe("Setup", () => {
  // Different than normal beforeEach because we don't run the fleetctl setup.
  beforeEach(() => {
    const SHELL = Cypress.platform === "win32" ? "cmd" : "bash";
    cy.exec("make e2e-reset-db", {
      timeout: 20000,
      env: { SHELL },
    });
  });

  it("Completes setup", () => {
    cy.visit("/");
    cy.url().should("match", /\/setup$/);
    cy.contains(/setup/i);

    fillOutForm();

    cy.url().should("match", /\/hosts\/manage$/i);
    cy.contains(/add your devices/i);
  });

  it("shows messaging when there is an error during setup", () => {
    cy.visit("/");
    cy.intercept("POST", "/api/v1/setup", { forceNetworkError: true });

    fillOutForm();

    cy.findByText(
      "We were unable to configure Fleet. If your Fleet server is behind a proxy, please ensure the server can be reached."
    ).should("exist");
  });
});

import dashboardPage from "../../pages/dashboardPage";

describe("Dashboard", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
  });

  after(() => {
    cy.logout();
  });

  describe("Operating systems card", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      dashboardPage.visitsDashboardPage();
    });

    it("displays operating systems card if macOS platform is selected", () => {
      dashboardPage.switchesPlatform("macOS");
    });

    it("displays operating systems card if Windows platform is selected", () => {
      dashboardPage.switchesPlatform("Windows");
    });

    it("displays operating systems card if Linux platform is selected", () => {
      dashboardPage.switchesPlatform("Linux");
    });
  });
  describe("Hosts filter by dashboard host summary", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/dashboard");
    });
    it("filters macOS hosts", () => {
      cy.findByText(/macos hosts/i).click();
      cy.findByRole("status", {
        name: /hosts filtered by macos/i,
      }).should("exist");
    });
    it("filters Windows hosts", () => {
      cy.findByText(/windows hosts/i).click();
      cy.findByRole("status", {
        name: /hosts filtered by windows/i,
      }).should("exist");
    });

    it("filters linux hosts", () => {
      cy.findByText(/macos hosts/i).click();
      cy.findByRole("status", {
        name: /hosts filtered by macos/i,
      }).should("exist");
    });
    // filters missing hosts and low disk space hosts on premium only, premium/admin.spec.ts
  });
});

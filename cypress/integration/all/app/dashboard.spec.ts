import AppSettingsPage from "../../../pages/appSettingsPage";

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
      cy.visit("/dashboard");
    });

    it("displays operating systems card if macOS platform is selected", () => {
      cy.getAttached(".homepage__platform_dropdown").click();
      cy.getAttached(".Select-menu-outer").within(() => {
        cy.findAllByText("macOS").click();
      });
      cy.getAttached(".operating-systems").should("exist");
    });

    it("displays operating systems card if Windows platform is selected", () => {
      cy.getAttached(".homepage__platform_dropdown").click();
      cy.getAttached(".Select-menu-outer").within(() => {
        cy.findAllByText("Windows").click();
      });
      cy.getAttached(".operating-systems").should("exist");
    });
  });

  describe("Activity Card", () => {
    beforeEach(() => {
      cy.loginWithCySession();
    });

    it("displays activity when editing agent options", () => {
      AppSettingsPage.visitAgentOptions();
      AppSettingsPage.editAgentOptionsForm("test:");

      cy.visit("/dashboard");

      // the edit agent options is split across multiple elements so we use a
      // matcher function and assert the different parts individually.
      cy.findByText("edited agent options.")
        .should("exist")
        .and("contain", "Admin");
    });
  });
});

describe("Dashboard)", () => {
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
  });
});

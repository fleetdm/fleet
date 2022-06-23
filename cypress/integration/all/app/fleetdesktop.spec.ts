const fakeDeviceToken = "phAK3d3vIC37OK3n";

describe("Fleet Desktop", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.addDockerHost();
    cy.setDesktopToken(1, fakeDeviceToken);
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.stopDockerHost();
  });
  describe("Fleet Desktop device user page", () => {
    beforeEach(() => {
      Cypress.session.clearAllSavedSessions();
    });
    it("serves device user page if url includes valid token", () => {
      cy.visit(`/device/${fakeDeviceToken}`);
      cy.findByText(/my device/i).should("exist");
    });
  });
});

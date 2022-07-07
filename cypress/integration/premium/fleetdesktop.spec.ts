const fakeDeviceToken = "phAK3d3vIC37OK3n";

describe("Fleet Desktop", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.addDockerHost();
    cy.setDesktopToken(1, fakeDeviceToken);
    cy.viewport(1200, 660);
    cy.seedPolicies();
  });
  after(() => {
    cy.stopDockerHost();
  });
  describe("Fleet Desktop device user page", () => {
    beforeEach(() => {
      Cypress.session.clearAllSavedSessions();
      cy.visit(`/device/${fakeDeviceToken}`);
    });
    it("renders policies and provides instructions for self-remediation", () => {
      cy.getAttached(".react-tabs__tab-list").within(() => {
        cy.findByText(/policies/i).click();
      });
      cy.getAttached(".section--policies").within(() => {
        cy.findByText(/is filevault enabled/i).click();
      });
      cy.getAttached(".policy-details-modal").within(() => {
        cy.findByText(/click turn on filevault/i).should("exist");
      });
    });
  });
});

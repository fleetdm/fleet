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
      cy.visit(`/device/${fakeDeviceToken}`);
    });
    it("renders the device user information and info modal", () => {
      cy.findByText(/my device/i).should("exist");
      cy.getAttached(".status--online").should("exist");
      cy.getAttached(".info-flex").within(() => {
        cy.findByText(/ubuntu 20/i)
          .prev()
          .contains(/operating system/i);
      });
      cy.getAttached(".info-grid").within(() => {
        cy.findByText(/private ip address/i)
          .next()
          .findByText(/---/i)
          .should("not.exist");
      });
      cy.getAttached(".device-user__action-button-container").within(() => {
        cy.getAttached('img[alt="Host info icon"]').click();
      });
      cy.getAttached(".device-user-info__modal").within(() => {
        cy.findByRole("button", { name: /ok/i }).click();
      });
    });
    it("renders and searches the host's software", () => {
      cy.getAttached(".react-tabs__tab-list").within(() => {
        cy.findByText(/software/i).click();
      });
      let initialCount = 0;
      cy.getAttached(".section--software").within(() => {
        cy.getAttached(".table-container__results-count")
          .invoke("text")
          .then((text) => {
            const fullText = text;
            const pattern = /[0-9]+/g;
            const newCount = fullText.match(pattern);
            initialCount = parseInt(newCount[0], 10);
            expect(initialCount).to.be.at.least(1);
          });
        cy.findByPlaceholderText(/search software/i).type("lib");
        // Ensures search completes
        cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
        cy.getAttached(".table-container__results-count")
          .invoke("text")
          .then((text) => {
            const fullText = text;
            const pattern = /[0-9]+/g;
            const newCount = fullText.match(pattern);
            const searchCount = parseInt(newCount[0], 10);
            expect(searchCount).to.be.lessThan(initialCount);
          });
      });
    });
    it(
      "refetches host vitals",
      {
        retries: {
          runMode: 2,
        },
        defaultCommandTimeout: 15000,
      },
      () => {
        cy.getAttached(".hostname-container").within(() => {
          cy.contains("button", /refetch/i).click();
          cy.findByText(/fetching/i).should("exist");
          cy.contains("button", /refetch/i).should("exist");
          cy.findByText(/less than a minute/i).should("exist");
        });
      }
    );
  });
});

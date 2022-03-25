describe("Device user only", () => {
  beforeEach(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
  });
  describe("Device user page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/device/deviceauthtoken"); // TODO: Lucas will create a script using token: deviceauthtoken
    });
    it("renders the device user information and info modal", () => {
      cy.getAttached(".status--online").should("exist");
      cy.getAttached(".info-flex").within(() => {
        cy.findByText(/operating system/i)
          .next()
          .contains(/operating system goes here/i); // TODO
      });
      cy.getAttached(".info-grid").within(() => {
        cy.findByText(/internal ip address/i)
          .next()
          .contains(/./i); // TODO
      });
      cy.getAttached(".device-user__action-button-container").within(() => {
        cy.getAttached('img[alt="Host info icon"]').click();
      });
      cy.getAttached(".device-user-info__modal").within(() => {
        cy.getAttached(".device-user-info__btn").click();
      });
    });
    it("renders and searches the host's software,  links to filter hosts by software", () => {
      cy.getAttached(".react-tabs__tab-list").within(() => {
        cy.findByText(/software/i).click();
      });
      let initialCount = 0;
      cy.getAttached(".section--software").within(() => {
        // TODO: Lucas creating a device which will include software
        cy.getAttached(".table-container__results-count")
          .invoke("text")
          .then((text) => {
            const fullText = text;
            const pattern = /[0-9]+/g;
            const newCount = fullText.match(pattern);
            initialCount = parseInt(newCount[0], 10);
            expect(initialCount).to.be.at.least(1);
          });
        cy.findByPlaceholderText(/filter software/i).type("lib");
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

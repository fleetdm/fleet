import _ = require("cypress/types/lodash");
import * as path from "path";

let hostname = "";

describe("Hosts flow", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.addDockerHost();
    cy.clearDownloads();
    cy.seedQueries();
    cy.seedPolicies();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
    cy.stopDockerHost();
  });
  describe("Manage hosts page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/hosts/manage");
    });
    it("adds a new host", () => {
      // Download installer
      cy.visit("/hosts/manage");

      cy.getAttached(".manage-hosts").within(() => {
        cy.contains("button", /generate installer/i).click();
      });

      cy.getAttached(".react-tabs").within(() => {
        cy.findByText(/rpm/i).should("exist").click();
      });

      cy.contains("a", /download/i)
        .first()
        .click();

      // Assert enroll secret downloaded matches the one displayed
      // NOTE: This test often fails when the Cypress downloads folder was not cleared properly
      // before each test run (seems to be related to issues with Cypress trashAssetsBeforeRun)
      if (Cypress.platform !== "win32") {
        // windows has issues with downloads location
        cy.readFile(path.join(Cypress.config("downloadsFolder"), "fleet.pem"), {
          timeout: 5000,
        });
      }
    });
  });
  describe("Manage policies page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/hosts/manage");
    });
    it(
      "runs policy on an existing host",
      {
        retries: {
          runMode: 2,
        },
        defaultCommandTimeout: 10000,
      },
      () => {
        cy.getAttached("tbody").within(() => {
          cy.get(".button--text-link").first().as("hostLink");
        });
        cy.getAttached("@hostLink")
          // Set hostname variable for later assertions
          .then((el) => {
            console.log(el);
            hostname = el.text();
            return el;
          })
          .click();
        // Go to host details page
        cy.location("pathname").should("match", /hosts\/[0-9]/i);
        cy.getAttached(".status--online").should("exist");
        // Run policy on host
        let policyname = "";
        cy.contains("a", "Policies").click();
        cy.getAttached("tbody").within(() => {
          cy.get(".button--text-link").first().as("policyLink");
        });
        cy.getAttached("@policyLink")
          // Set policyname variable for later assertions
          .then((el) => {
            console.log(el);
            policyname = el.text();
            return el;
          });
        cy.findByText(/filevault/i)
          .should("exist")
          .click();
        cy.findByText(/run/i).should("exist").click();
        cy.findByText(/all hosts/i)
          .should("exist")
          .click()
          .then(() => {
            cy.findByText(/run/i).click();
          });
        cy.getAttached(".data-table").within(() => {
          cy.findByText(hostname).should("exist");
        });
      }
    );
  });
  describe("Host details page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/hosts/manage");
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".button--text-link").first().click();
      });
    });
    it(
      "runs query on an existing host",
      {
        retries: {
          runMode: 2,
        },
        defaultCommandTimeout: 10000,
      },
      () => {
        cy.getAttached(".host-details__action-button-container").within(() => {
          cy.getAttached('img[alt="Query host icon"]').click();
        });

        cy.getAttached(".select-query-modal__modal").within(() => {
          cy.getAttached(".modal-query-button").eq(2).click();
        });

        cy.getAttached(".query-form__button-wrap--new-query").within(() => {
          cy.findByText(/run query/i)
            .should("exist")
            .click();
        });
        cy.getAttached(".query-page__wrapper").within(() => {
          cy.getAttached(".data-table").within(() => {
            cy.findByText(hostname).should("exist");
          });
          cy.findByText(/run/i).click();
        });
      }
    );
    it("deletes an existing host", () => {
      cy.getAttached(".host-details__action-button-container")
        .within(() => {
          cy.findByText(/delete/i).click();
        })
        .then(() => {
          cy.getAttached(".modal__modal_container")
            .within(() => {
              cy.findByText(/delete host/i).should("exist");
              cy.findByRole("button", { name: /delete/i }).click();
            })
            .then(() => {
              cy.findByText(/add your devices to fleet/i).should("exist");
              cy.findByText(/generate installer/i).should("exist");
              cy.findByText(/about this host/i).should("not.exist");
              cy.findByText(hostname).should("not.exist");
            });
        });
    });
  });
});

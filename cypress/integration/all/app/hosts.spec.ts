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
    // TODO: cy.seedSchedule();
    cy.viewport(1200, 660);
  });
  after(() => {
    // cy.logout();
    // cy.stopDockerHost();
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
        cy.findByText(/rpm/i).first().should("exist").click();
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
    it(
      "renders and searches the host's users",
      {
        retries: {
          runMode: 2,
        },
        defaultCommandTimeout: 10000,
      },
      () => {
        cy.getAttached(".section--users").within(() => {
          cy.getAttached("tbody>tr").should("have.length.greaterThan", 0);
          cy.findByPlaceholderText(/search/i).type("Ash");
          cy.getAttached("tbody>tr").should("have.length", 0);
          cy.getAttached(".empty-users").within(() => {
            cy.findByText(/no users matched/i).should("exist");
          });
        });
      }
    );
    it(
      "renders and searches the host's software,  links to filter hosts by software",
      {
        retries: {
          runMode: 2,
        },
        defaultCommandTimeout: 10000,
      },
      () => {
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
              initialCount = parseInt(newCount[0]);
              console.log("softwareCount", initialCount);
              expect(initialCount).to.be.at.least(1);
            });
          cy.findByPlaceholderText(/filter software/i).type("lib");
          console.log("softwareCount", initialCount);
          // Ensures search completes
          cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
          cy.getAttached(".table-container__results-count")
            .invoke("text")
            .then((text) => {
              const fullText = text;
              const pattern = /[0-9]+/g;
              const newCount = fullText.match(pattern);
              const searchCount = parseInt(newCount[0]);
              expect(searchCount).to.be.lessThan(initialCount);
            });
          cy.getAttached(".software-link").first().click({ force: true });
          cy.getAttached(".manage-hosts__software-filter-block").within(() => {
            cy.getAttached(".manage-hosts__software-filter-name-card").should(
              "exist"
            );
            cy.getAttached(".data-table").within(() => {
              cy.findByText(hostname).should("exist");
            });
          });
          cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
          cy.getAttached(".manage-hosts__software-filter-block").within(() => {
            cy.getAttached(".manage-hosts__software-filter-name-card").should(
              "exist"
            );
            cy.getAttached(".data-table").within(() => {
              cy.findByText(hostname).should("exist");
            });
          });
        });
      }
    );
    it(
      "renders host's schedule",
      {
        retries: {
          runMode: 2,
        },
        defaultCommandTimeout: 10000,
      },
      () => {
        cy.getAttached(".react-tabs__tab-list").within(() => {
          cy.findByText(/schedule/i).click();
        });
        cy.getAttached(".data-table").within(() => {
          cy.findByText(/query name/i).should("exist");
        });
      }
    );
    it(
      "renders host's policies and links to filter hosts by policy status",
      {
        retries: {
          runMode: 2,
        },
        defaultCommandTimeout: 10000,
      },
      () => {
        cy.getAttached(".react-tabs__tab-list").within(() => {
          cy.findByText(/policies/i).click();
        });
        cy.getAttached(".section--policies").within(() => {
          cy.findByText(/failing 1 policy/i).should("exist");
          cy.getAttached(".policy-link").first().click({ force: true });
          cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
          cy.getAttached(".manage-hosts__policies-filter-name-card").should(
            "exist"
          );
        });
      }
    );
    it(
      "refetches host vitals",
      {
        retries: {
          runMode: 2,
        },
        defaultCommandTimeout: 10000,
      },
      () => {
        cy.getAttached(".hostname-container").within(() => {
          cy.contains("button", /refetch/i).click();
          cy.findByText(/fetching/i).should("exist");
          // Ensures fetching completes
          cy.wait(5000); // eslint-disable-line cypress/no-unnecessary-waiting
          cy.contains("button", /refetch/i).should("exist");
          cy.findByText(/few seconds/i).should("exist");
        });
      }
    );
  });
});

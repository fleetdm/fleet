import * as path from "path";
import { format } from "date-fns";
import manageHostsPage from "../../pages/manageHostsPage";
import hostDetailsPage from "../../pages/hostDetailsPage";

let hostname = "";

describe("Hosts flow", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.addDockerHost();
    cy.clearDownloads();
    cy.seedQueries();
    cy.seedSchedule();
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
      manageHostsPage.visitsManageHostsPage();
    });
    it("adds a new host and downloads installation files", () => {
      // Download add hosts files
      cy.getAttached(".manage-hosts").within(() => {
        cy.contains("button", /add hosts/i).click();
      });
      cy.getAttached(".react-tabs").within(() => {
        cy.findByText(/advanced/i)
          .first()
          .should("exist")
          .click();
      });
      cy.getAttached(".reveal-button").click();
      cy.getAttached('a[href*="#downloadEnrollSecret"]').click();
      cy.getAttached('a[href*="#downloadCertificate"]').last().click();
      cy.getAttached('a[href*="#downloadFlagfile"]').click();

      // NOTE: This test often fails when the Cypress downloads folder was not cleared properly
      // before each test run (seems to be related to issues with Cypress trashAssetsBeforeRun)
      if (Cypress.platform !== "win32") {
        // windows has issues with downloads location

        // Feature pushed back from 4.13 release
        // const formattedTime = format(new Date(), "yyyy-MM-dd");
        // const filename = `Hosts ${formattedTime}.csv`;
        // cy.readFile(path.join(Cypress.config("downloadsFolder"), filename), {
        //   timeout: 5000,
        // });
        cy.readFile(
          path.join(Cypress.config("downloadsFolder"), "secret.txt"),
          {
            timeout: 5000,
          }
        );
        cy.readFile(
          path.join(Cypress.config("downloadsFolder"), "flagfile.txt"),
          {
            timeout: 5000,
          }
        );
        cy.readFile(path.join(Cypress.config("downloadsFolder"), "fleet.pem"), {
          timeout: 5000,
        });
      }
    });
    it(`exports hosts to CSV`, () => {
      cy.getAttached(".manage-hosts").within(() => {
        cy.getAttached(".manage-hosts__export-btn").click();
      });
      if (Cypress.platform !== "win32") {
        // windows has issues with downloads location
        const formattedTime = format(new Date(), "yyyy-MM-dd");
        const filename = `Hosts ${formattedTime}.csv`;
        cy.readFile(path.join(Cypress.config("downloadsFolder"), filename), {
          timeout: 5000,
        });
      }
    });
    it(`hides and shows "Used by" column`, () => {
      cy.getAttached("thead").within(() =>
        cy.findByText(/used by/i).should("not.exist")
      );
      cy.getAttached(".table-container").within(() => {
        cy.contains("button", /edit columns/i).click();
      });
      cy.getAttached(".edit-columns-modal").within(() => {
        cy.findByLabelText(/used by/i).check({ force: true });
        cy.contains("button", /save/i).click();
      });
      cy.getAttached("thead").within(() =>
        cy.findByText(/used by/i).should("exist")
      );
    });
  });
  describe("Manage policies page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      manageHostsPage.visitsManageHostsPage();
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
            hostname = el.text();
            return el;
          })
          .click();
        // Go to host details page
        cy.location("pathname").should("match", /hosts\/[0-9]/i);
        cy.getAttached(".status--online").should("exist");
        // Run policy on host
        cy.contains("a", "Policies").click();
        cy.getAttached("tbody").within(() => {
          cy.get(".button--text-link").first().as("policyLink");
        });
        cy.getAttached("@policyLink")
          // Set policyname variable for later assertions
          .then((el) => {
            console.log(el);
            return el;
          });
        cy.findByText(/filevault/i)
          .should("exist")
          .click();
        cy.getAttached(".policy-form__run").should("exist").click();
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
      manageHostsPage.visitsManageHostsPage();
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".button--text-link").first().click();
      });
    });
    it("renders and searches the host's users", () => {
      cy.getAttached(".section--users").within(() => {
        cy.getAttached("tbody>tr").should("have.length.greaterThan", 0);
        cy.findByPlaceholderText(/search/i).type("Ash");
        cy.getAttached("tbody>tr").should("have.length", 0);
        cy.getAttached(".empty-table__container").within(() => {
          cy.findByText(/no users match/i).should("exist");
        });
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
    it("host's software table links to filter hosts by software", () => {
      cy.getAttached(".react-tabs__tab-list").within(() => {
        cy.findByText(/software/i).click();
      });
      cy.getAttached(".software-link").first().click({ force: true });
      cy.findByText(/adduser 3.118ubuntu2/i).should("exist"); // first seeded software item
      cy.getAttached(".data-table").within(() => {
        cy.findByText(hostname).should("exist");
      });
    });
    it("host's software table links to software details", () => {
      cy.getAttached(".react-tabs__tab-list").within(() => {
        cy.findByText(/software/i).click();
      });
      cy.contains(/adduser/i).click();
      cy.findByText(/adduser, 3.118ubuntu2/i).should("exist");
    });
    it("renders host's schedule", () => {
      cy.getAttached(".react-tabs__tab-list").within(() => {
        cy.findByText(/schedule/i).click();
      });
      cy.getAttached(".data-table").within(() => {
        cy.getAttached(".query_name__header").should("exist");
      });
    });
    it("renders host's policies and links to filter hosts by policy status", () => {
      cy.getAttached(".react-tabs__tab-list").within(() => {
        cy.findByText(/policies/i).click();
      });
      cy.getAttached(".section--policies").within(() => {
        cy.findByText(/failing 1 policy/i).should("exist");
        cy.getAttached(".policy-link").first().click({ force: true });
      });
      cy.findAllByText(/Is Filevault enabled on macOS devices/i).should(
        "exist"
      );
      cy.getAttached(".data-table").within(() => {
        cy.findByText(hostname).should("exist");
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
        cy.getAttached(".display-name-container").within(() => {
          cy.contains("button", /refetch/i).click();
          cy.findByText(/fetching/i).should("exist");
          cy.contains("button", /refetch/i).should("exist");
          cy.findByText(/less than a minute/i).should("exist");
        });
      }
    );
  });
});

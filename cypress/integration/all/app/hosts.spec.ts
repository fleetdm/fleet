import * as path from "path";

describe(
  "Hosts flow",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.addDockerHost();
      cy.clearDownloads();
      cy.seedQueries();
      cy.seedPolicies();
    });

    afterEach(() => {
      cy.stopDockerHost();
    });

    it(
      "Add new host, run policy on host, and delete a host successfully",
      {
        retries: {
          runMode: 2,
        },
      },
      () => {
        let hostname = "";

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
          cy.readFile(
            path.join(Cypress.config("downloadsFolder"), "fleet.pem"),
            {
              timeout: 5000,
            }
          );
        }

        cy.visit("/hosts/manage");

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
            cy.get(".data-table").within(() => {
              cy.findByText(hostname).should("exist");
            });
          });

        cy.visit("/hosts/manage");

        cy.getAttached("tbody").within(() => {
          cy.get(".button--text-link").first().as("hostLink");
        });

        // Select a query to run on a specified host on host details page
        cy.getAttached("@hostLink")
          .click()
          .then(() => {
            cy.findByText(/about this host/i).should("exist");
            cy.findByText(hostname).should("exist");

            cy.getAttached('img[alt="Query host icon"]').click();
            cy.getAttached(".modal__modal_container")
              .within(() => {
                cy.findByText(/select a query/i).should("exist");
                cy.findByText(/detect presence/i).click();
              })
              .then(() => {
                cy.findByText(/run query/i).click();
                cy.getAttached(".data-table").within(() => {
                  cy.findByText(hostname).should("exist");
                });
              });
          });

        // Delete host
        cy.visit("/hosts/manage");

        cy.getAttached("@hostLink")
          .click()
          .then(() => {
            cy.get('img[alt="Delete host icon"]').click();
            cy.get(".modal__modal_container")
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
      }
    );
  }
);

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
      "Can add new host from manage hosts page, run policy on host, and delete a host",
      {
        retries: {
          runMode: 2,
        },
      },
      () => {
        let hostname = "";
        cy.visit("/hosts/manage");
        cy.get(".manage-hosts").should("contain", /hostname/i); // Ensures page load

        cy.contains("button", /generate installer/i).click();
        cy.findByText(/rpm/i).should("exist").click();
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
        cy.location("pathname").should("match", /hosts\/manage/i);
        cy.get(".manage-hosts").should("contain", /hostname/i); // Ensures page load

        cy.get("tbody").within(() => {
          cy.get(".button--text-link").first().as("hostLink");
        });

        cy.get("@hostLink")
          // Set hostname variable for later assertions
          .then((el) => {
            console.log(el);
            hostname = el.text();
            return el;
          })
          .click();

        // Go to host details page
        cy.location("pathname").should("match", /hosts\/[0-9]/i);
        cy.get("span.status").should("contain", /online/i);

        // Run policy on host
        let policyname = "";
        cy.contains("a", "Policies").click();
        cy.wait(2000); // Ensuring page load with table is flakey, temp solution wait
        // cy.get(".table-container").should("contain", /filevault/i); // Ensures page load

        cy.get("tbody").within(() => {
          cy.get(".button--text-link").first().as("policyLink");
        });

        cy.get("@policyLink")
          // Set policyname variable for later assertions
          .then((el) => {
            console.log(el);
            policyname = el.text();
            return el;
          });

        cy.findByText(/filevault/i)
          .should("exist")
          .click();

        cy.findByText(/run/i).should("exist").click(); // Ensures page load

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
        cy.get(".manage-hosts").should("contain", /hostname/i); // Ensures page load

        cy.get("tbody").within(() => {
          cy.get(".button--text-link").first().as("hostLink");
        });

        cy.get("@hostLink")
          .click()
          .then(() => {
            cy.findByText(/about this host/i).should("exist");
            cy.findByText(hostname).should("exist");

            // Open query host modal and select query
            cy.get('img[alt="Query host icon"]').click();
            cy.get(".modal__modal_container")
              .within(() => {
                cy.findByText(/select a query/i).should("exist");
                cy.findByText(/detect presence/i).click();
              })
              .then(() => {
                cy.findByText(/run query/i).click();
                cy.get(".data-table").within(() => {
                  cy.findByText(hostname).should("exist");
                });
              });
          });

        cy.visit("/hosts/manage");
        cy.get(".manage-hosts").should("contain", /hostname/i); // Ensures page load

        cy.get("@hostLink")
          .click()
          .then(() => {
            // Open delete host modal and delete host
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

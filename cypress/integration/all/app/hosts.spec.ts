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
    });

    afterEach(() => {
      cy.stopDockerHost();
    });

    it(
      "Can add new host from manage hosts page",
      {
        retries: {
          runMode: 2,
        },
      },
      () => {
        cy.visit("/hosts/manage");

        cy.contains("button", /generate installer/i).click();
        // eslint-disable-next-line cypress/no-unnecessary-waiting
        cy.wait(2000);
        cy.findByText(/rpm/i).click();
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
        // eslint-disable-next-line cypress/no-unnecessary-waiting
        cy.wait(1000);
        cy.get('button[title="Online"]').click();

        // Go to host details page
        cy.location("pathname").should("match", /hosts\/[0-9]/i);
        // eslint-disable-next-line cypress/no-unnecessary-waiting
        cy.wait(1000);
        cy.get("span.status").contains(/online/i);
      }
    );

    it("Can query a host and delete a host from host details page", () => {
      let hostname = "";

      cy.visit("/hosts/manage");

      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(3000);

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
        .click()
        .then(() => {
          cy.findByText(/about this host/i).should("exist");
          cy.findByText(hostname).should("exist");

          // Open query host modal and cancel
          cy.get('img[alt="Query host icon"]').click();
          cy.get(".modal__modal_container").within(() => {
            cy.findByText(/select a query/i).should("exist");
            cy.get(".modal__ex").click();
          });
          cy.findByText(/select a query/i).should("not.exist");

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

      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(3000);
      cy.get("@hostLink")
        // Set hostname variable for later assertions
        .then((el) => {
          console.log(el);
          hostname = el.text();
          return el;
        })
        .click()
        .then(() => {
          // Open delete host modal and cancel
          cy.get('img[alt="Delete host icon"]').click();
          cy.get(".modal__modal_container").within(() => {
            cy.findByText(/delete host/i).should("exist");
            cy.findByRole("button", { name: /cancel/i }).click();
          });
          cy.findByText(/delete host/i).should("not.exist");

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
    });
  }
);

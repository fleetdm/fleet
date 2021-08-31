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
        cy.visit("/");

        cy.contains("button", /add new host/i).click();

        cy.get('a[href*="showSecret"]').click();
        cy.contains("a", /download/i)
          .first()
          .click();

        // Assert enroll secret downloaded matches the one displayed
        // NOTE: This test often fails when the Cypress downloads folder was not cleared properly
        // before each test run (seems to be related to issues with Cypress trashAssetsBeforeRun)
        cy.readFile(
          path.join(Cypress.config("downloadsFolder"), "secret.txt"),
          {
            timeout: 5000,
          }
        ).then((contents) => {
          cy.get("input[disabled]").should("have.value", contents);
        });

        // Wait until the host becomes available (usually immediate in local
        // testing, but may vary by environment).
        cy.waitUntil(
          () => {
            cy.visit("/");
            return Cypress.$('button[title="Online"]').length > 0;
          },
          { timeout: 30000, interval: 1000 }
        );

        // Go to host details page
        cy.get('button[title="Online"]').click();
        cy.get("span.status").contains("online");
      }
    );

    // Test commented out
    // Pending fix to prevent consistent failing in GitHub

    // it("Can delete a host from host details page", () => {
    //   let hostname = "";

    //   cy.visit("/hosts/manage");

    //   cy.get("tbody").within(() => {
    //     cy.get(".button--text-link").first().as("hostLink");
    //   });

    //   cy.get("@hostLink")
    //     // Set hostname variable for later assertions
    //     .then((el) => {
    //       console.log(el);
    //       hostname = el.text();
    //       return el;
    //     })
    //     .click()
    //     .then(() => {
    //       cy.findByText(/about this host/i).should("exist");
    //       cy.findByText(hostname).should("exist");

    //       // Open delete host modal and cancel
    //       cy.get('img[alt="Delete host icon"]').click();
    //       cy.get(".modal__modal_container").within(() => {
    //         cy.findByText(/delete host/i).should("exist");
    //         cy.findByRole("button", { name: /cancel/i }).click();
    //       });
    //       cy.findByText(/delete host/i).should("not.exist");

    //       // Open delete host modal and delete host
    //       cy.get('img[alt="Delete host icon"]').click();
    //       cy.get(".modal__modal_container")
    //         .within(() => {
    //           cy.findByText(/delete host/i).should("exist");
    //           cy.findByRole("button", { name: /delete/i }).click();
    //         })
    //         .then(() => {
    //           cy.findByText(/successfully deleted/i).should("exist");
    //           cy.findByText(/kinda empty in here/i).should("exist");
    //           cy.findByText(/about this host/i).should("not.exist");
    //           cy.findByText(hostname).should("not.exist");
    //         });
    //     });
    // });

    // Test commented out
    // Pending fix to prevent consistent failing in GitHub

    // it("Can query a host from host details page", () => {
    //   cy.seedQueries();

    //   let hostname = "";

    //   cy.visit("/hosts/manage");

    //   cy.get("tbody").within(() => {
    //     cy.get(".button--text-link").first().as("hostLink");
    //   });

    //   cy.get("@hostLink")
    //     // Set hostname variable for later assertions
    //     .then((el) => {
    //       hostname = el.text();
    //       return el;
    //     })
    //     .click()
    //     .then(() => {
    //       cy.findByText(/about this host/i).should("exist");
    //       cy.findByText(hostname).should("exist");

    //       // Open query host modal and cancel
    //       cy.get('img[alt="Query host icon"]').click();
    //       cy.get(".modal__modal_container").within(() => {
    //         cy.findByText(/select a query/i).should("exist");
    //         cy.get(".modal__ex").click();
    //       });
    //       cy.findByText(/select a query/i).should("not.exist");

    //       // Open query host modal and select query
    //       cy.get('img[alt="Query host icon"]').click();
    //       cy.get(".modal__modal_container")
    //         .within(() => {
    //           cy.findByText(/select a query/i).should("exist");
    //           cy.findByText(/detect presence/i).click();
    //         })
    //         .then(() => {
    //           cy.findByText(/edit & run query/i).should("exist");
    //           cy.get(".target-select").within(() => {
    //             cy.findByText(hostname).should("exist");
    //           });
    //         });
    //     });
    // });
  }
);

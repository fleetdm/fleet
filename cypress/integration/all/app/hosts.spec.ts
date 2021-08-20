import * as path from "path";

describe("Hosts page", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.addDockerHost();
  });

  afterEach(() => {
    cy.stopDockerHost();
  });

  it(
    "Add new host",
    {
      retries: {
        runMode: 2,
      },
    },
    () => {
      cy.visit("/");

      cy.contains("button", /add new host/i).click();

      cy.get(".add-host-modal__secret-wrapper").within(() => {
        // Check if select team dropdown is present
        if (Cypress.$(".add-host-modal__team-dropdown-wrapper").length) {
          cy.get(".Select-placeholder").click();
          cy.get(".dropdown__option").first().click(); // click "No team" option in dropdown
        }
        cy.get('a[href*="showSecret"]').click();
        cy.contains("a", /download/i)
          .first()
          .click();
      });

      // Assert enroll secret downloaded matches the one displayed
      // NOTE: This test often fails when the Cypress downloads folder was not cleared properly
      // before each test run (seems to be related to issues with Cypress trashAssetsBeforeRun)
      cy.readFile(path.join(Cypress.config("downloadsFolder"), "secret.txt"), {
        timeout: 5000,
      }).then((contents) => {
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
});

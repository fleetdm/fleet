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

      cy.contains("a", /download/i)
        .first()
        .click();

      cy.get('a[href*="showSecret"]').click();

      // Assert enroll secret downloaded matches the one displayed
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

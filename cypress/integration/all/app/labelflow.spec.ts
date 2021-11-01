describe(
  "Label flow",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
    });

    it("Create, edit, and delete a label successfully", () => {
      cy.visit("/hosts/manage");

      cy.findByRole("button", { name: /add label/i }).click();

      // Using class selector because third party element doesn't work with Cypress Testing Selector Library
      cy.get(".ace_content")
        .click()
        .type("{selectall}{backspace}SELECT * FROM users;");

      cy.findByLabelText(/name/i).click().type("Show all users");

      cy.findByLabelText(/description/i)
        .click()
        .type("Select all users across platforms.");

      // =========
      // TODO: Not needed if we're selecting "All platforms"
      // either choose a selection or leave it - otherwise
      // blocks the "Save label" button and breaks the test

      // Cannot call cy.select on div disguised as a dropdown
      // cy.findByText(/all platforms/i).click();

      cy.findByRole("button", { name: /save label/i }).click();
      // =========

      // edit custom label
      cy.findByText(/show all users/i).click();

      cy.get(".manage-hosts__label-block button").first().click();

      // Label SQL not editable to test

      cy.findByLabelText(/name/i)
        .click()
        .type("{selectall}{backspace}Show all usernames");

      cy.findByLabelText(/description/i)
        .click()
        .type("{selectall}{backspace}Select all usernames on Mac.");

      cy.findByText(/select one/i).should("not.exist");

      cy.findByRole("button", { name: /update label/i }).click();

      // Close success notification
      cy.get(".flash-message__remove").click();

      cy.visit("/hosts/manage");

      cy.wait(3000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.findByText(/show all usernames/i).click();

      // delete custom label
      cy.get(".manage-hosts__label-block button").last().click();

      cy.wait(4000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.get(".manage-hosts__modal-buttons > .button--alert")
        .contains("button", /delete/i)
        .click();

      cy.findByText(/show all users/i).should("not.exist");
    });
  }
);

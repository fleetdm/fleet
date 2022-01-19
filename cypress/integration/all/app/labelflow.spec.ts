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
      cy.getAttached(".ace_content").type(
        "{selectall}{backspace}SELECT * FROM users;"
      );

      cy.findByLabelText(/name/i).click().type("Show all MAC users");

      cy.findByLabelText(/description/i)
        .click()
        .type("Select all MAC users.");

      cy.getAttached(".label-form__form-field--platform > .Select").click();

      cy.getAttached(".Select-menu-outer").within(() => {
        cy.findByText(/macOS/i).click();
      });

      cy.findByRole("button", { name: /save label/i }).click();

      // Edit custom label
      cy.getAttached(".host-side-panel").within(() => {
        cy.findByText(/show all mac users/i).click();
      });

      cy.getAttached(".manage-hosts__label-block button").first().click();

      // SQL and Platform are immutable fields

      cy.findByLabelText(/name/i).clear().type("Show all mac usernames");

      cy.findByLabelText(/description/i)
        .clear()
        .type("Select all usernames on Mac.");

      cy.findByText(/select one/i).should("not.exist");

      cy.findByRole("button", { name: /update label/i }).click();

      cy.getAttached(".host-side-panel").within(() => {
        cy.findByText(/show all mac usernames/i).click();
      });

      // Delete custom label
      cy.getAttached(".manage-hosts__label-block button").last().click();

      cy.getAttached(".manage-hosts__modal-buttons > .button--alert")
        .contains("button", /delete/i)
        .click();

      cy.getAttached(".host-side-panel").within(() => {
        cy.findByText(/show all mac usernames/i).should("not.exist");
      });
    });
  }
);

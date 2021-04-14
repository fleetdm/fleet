describe("Label flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
  });

  it("Create, edit, and delete a label successfully", () => {
    cy.visit("/hosts/manage");

    cy.findByRole("button", { name: /add new label/i }).click();

    // Using class selector because third party element doesn't work with Cypress Testing Selector Library
    cy.get(".ace_content")
      .click()
      .type("{selectall}{backspace}SELECT * FROM users;");

    cy.findByLabelText(/name/i).click().type("Show all users");

    cy.findByLabelText(/description/i)
      .click()
      .type("Select all users across platforms.");

    // Cannot call cy.select on div disguised as a dropdown
    cy.findByText(/select one/i).click();
    cy.findByText(/all platforms/i).click();

    cy.findByRole("button", { name: /save label/i }).click();

    cy.findByText(/show all users/i).click();

    cy.contains("button", /edit/i).click();

    // Label SQL not editable to test

    cy.findByLabelText(/name/i)
      .click()
      .type("{selectall}{backspace}Show all usernames");

    cy.findByLabelText(/description/i)
      .click()
      .type("{selectall}{backspace}Select all usernames on Mac.");

    cy.findByText(/select one/i).click();

    cy.findAllByText(/macos/i).click();

    cy.findByRole("button", { name: /update label/i }).click();

    cy.findByRole("button", { name: /delete/i }).click();

    // Can't figure out how attach findByRole onto modal button
    // Can't use findByText because delete button under modal
    cy.get(".manage-hosts__modal-buttons > .button--alert")
      .contains("button", /delete/i)
      .click();

    cy.findByText(/show all users/i).should("not.exist");
  });
});

describe("Manage Users", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.setupSMTP();
  });

  it("Searching for a user", () => {
    cy.visit("/settings/users");

    cy.findByText("admin@example.com").should("exist");
    cy.findByText("maintainer@example.com").should("exist");
    cy.findByText("observer@example.com").should("exist");
    cy.findByText("sso_user@example.com").should("exist");

    cy.findByPlaceholderText("Search").type("admin");

    cy.findByText("admin@example.com").should("exist");
    cy.findByText("maintainer@example.com").should("not.exist");
    cy.findByText("observer@example.com").should("not.exist");
    cy.findByText("sso_user@example.com").should("not.exist");
  });

  // it('Creating a user', () => {
  //   cy.visit('/settings/users');
  //   cy.url().should('match', /\/settings\/users$/i);
  //
  //   cy.contains('button:enabled', /create user/i)
  //     .click();
  //
  //   cy.findByPlaceholderText('Full name')
  //     .type('New User');
  //
  //   cy.findByPlaceholderText('Email')
  //     .type('new-user@example.com');
  //
  //   cy.findByRole('checkbox', { name: 'Test Team' })
  //     .click({ force: true }); // we use `force` as the checkbox button is not fully accessible yet.
  // });
});

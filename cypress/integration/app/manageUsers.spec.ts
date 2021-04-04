describe('Manage Users', () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.setupSMTP();
  });

  it('Searching for a user', () => {
    cy.intercept({
      method: 'GET',
      url: '/api/v1/fleet/users',
    }).as('getUsers');

    cy.visit('/settings/users');
    cy.url().should('match', /\/settings\/users$/i);

    // cy.wait('@getUsers');
    //
    // cy.findByText('test@fleetdm.com')
    //   .should('exist');
    // cy.findByText('test+1@fleetdm.com')
    //   .should('exist');
    // cy.findByText('test+2@fleetdm.com')
    //   .should('exist');
    //
    // cy.findByPlaceholderText('Search')
    //   .type('test@fleetdm.com');
    //
    // cy.wait('@getUsers');
    //
    // cy.findByText('test@fleetdm.com')
    //   .should('exist');
    // cy.findByText('test+1@fleetdm.com')
    //   .should('not.exist');
    // cy.findByText('test+2@fleetdm.com')
    //   .should('not.exist');
  });

  // it('Creating a user', () => {
  //   cy.visit('/settings/users');
  //   cy.url().should('match', /\/settings\/users$/i);
  //
  //   cy.contains('button:enabled', /create user/i)
  //     .click();
  //
  //   cy.findByPlaceholderText('Full Name')
  //     .type('New User');
  //
  //   cy.findByPlaceholderText('Email')
  //     .type('new-user@fleetdm.com');
  //
  //   cy.findByRole('checkbox', { name: 'Test Team' })
  //     .click({ force: true }); // we use `force` as the checkbox button is not fully accessible yet.
  // });
});

describe('Sessions', () => {
  before(() => {
    cy.exec('make e2e-reset-db e2e-setup')
  })
  
  it('Logs in and out successfully', () => {
    cy.visit('/');
    cy.contains(/forgot password/i);

    // Log in
    cy.get('input').first()
      .type('test@fleetdm.com');
    cy.get('input').last()
      .type('admin123#');
    cy.get('button')
      .click();

    // Verify dashboard
    cy.url().should('include', '/hosts/manage');
    cy.contains('All Hosts');

    // Log out
    cy.findByAltText(/user avatar/i)
      .click();
    cy.contains('button', 'Sign out')
      .click();

    cy.url().should('match', /\/login$/)
  });

  it('Fails login with invalid password', () => {
    cy.visit('/');
    cy.get('input').first()
      .type('test@fleetdm.com');
    cy.get('input').last()
      .type('bad_password');
    cy.get('.button')
      .click();

    cy.url().should('match', /\/login$/);
    cy.contains('username or email and password do not match');
  });
});

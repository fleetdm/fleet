describe('Login', () => {
  it('Logs in successfully', () => {
    cy.visit('/');
    cy.get(':nth-child(1) > .input-icon-field__input')
      .type('test@fleetdm.com');

    cy.get(':nth-child(2) > .input-icon-field__input')
      .type('admin123#');

    cy.get('.button')
      .click();

    cy.url().should('include', '/hosts/manage');
    cy.contains('All Hosts');
  });

  it('Fails with invalid password', () => {
    cy.visit('/');
    cy.get(':nth-child(1) > .input-icon-field__input')
      .type('test@fleetdm.com');

    cy.get(':nth-child(2) > .input-icon-field__input')
      .type('bad_password');

    cy.get('.button')
      .click();

    cy.url().should('include', '/login');
    cy.contains('username or email and password do not match');
  });
});

describe('Login', () => {
  beforeEach(() => {
    cy.exec('make e2e-reset-db e2e-setup')
  })
  
  it('Logs in successfully', () => {
    cy.visit('/');

    cy.contains('forgot password', { matchCase: false })
    
    cy.get('input').first()
      .click()
      .type('test@fleetdm.com');

    cy.get('input').last()
      .click().type('admin123#');

    cy.get('.button')
      .click();

    cy.url().should('include', '/hosts/manage');
    cy.contains('All Hosts');
  });

  it('Fails with invalid password', () => {
    cy.visit('/');
    cy.get('input').first()
      .click().type('test@fleetdm.com');

    cy.get('input').last()
      .click().type('bad_password');

    cy.get('.button')
      .click();

    cy.url().should('include', '/login');
    cy.contains('username or email and password do not match');
  });
});

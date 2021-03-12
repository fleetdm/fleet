describe('Setup', () => {
  beforeEach(() => {
    cy.exec('make e2e-reset-db')
  })
  
  it('Completes setup', () => {
    cy.visit('/');

    cy.url().should('include', '/setup');
      
    cy.contains('setup', { matchCase: false })

    // Page 1
    cy.findByPlaceholderText('Username')
      .type('test');

    cy.findByPlaceholderText('Password')
      .type('admin123#');

    cy.findByPlaceholderText('Confirm password')
      .type('admin123#');

    cy.findByPlaceholderText('Email')
      .type('test@fleetdm.com');

    cy.contains('button:enabled', 'Next')
      .click();

    // Page 2
    cy.findByPlaceholderText('Organization name')
      .type('Fleet Test')

    cy.contains('button:enabled', 'Next')
      .click();

    // Page 3
    cy.contains('button:enabled', 'Submit')
      .click();


    // Page 4
    // TODO figure out what is going on with the exception here.
    cy.on('uncaught:exception', () => { return false });
    cy.contains('button:enabled', 'Finish')
      .click();

      
    cy.url().should('contain', '/hosts/manage');
      
    cy.contains('All Hosts');
  });
});

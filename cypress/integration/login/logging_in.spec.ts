describe('Searching for a host', () => {
  it('Logs into the applications', () => {
    cy.visit('https://localhost:8080');
    cy.get(':nth-child(1) > .input-icon-field__input')
      .type('test@fleetdm.com');

    cy.get(':nth-child(2) > .input-icon-field__input')
      .type('admin123#');

    cy.get('.button')
      .click();

    cy.url().should('include', '/hosts/manage');
    cy.contains('All Hosts');
  });
});

describe('Hosts page', () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
  });
    
  it('Add new host', () => {
    cy.visit('/');

    cy.contains('button', /add new host/i)
      .click();

      cy.contains('a', /download/i);

      // TODO verify contents of downloads
  });
});

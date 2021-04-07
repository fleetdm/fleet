describe('Label flow', () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
    });

    // create, edit, delete, all one test

  it('Create a label successfully', () => {
    cy.visit('/hosts/manage');

    cy.contains('button', /add new label/i)
    .click();

    // fill in SQL
    // "select * from users;"

    // fill in Name
    // "Show Users"

    // fill in description
    // "Select all from users."


    // dropdown Platform
    // macOS
    //https://docs.cypress.io/api/commands/select

    // click save label
    cy.contains('button', /save label/i)
    .click();

  });

  it('Edit a label successfully', () => {

    cy.contains('button', /show users/i)
    .click();

    cy.contains('button', /edit/i)
    .click();

    // modify in SQL
    // "select * from users;"
    cy.get('input').______inputfieldvalue()
    .type('select * from users;');

    // modify in Name
    // "Show Users"

    // modify in description
    // "Select all from users."


    // modify dropdown Platform
    // macOS

    cy.contains('button', /update label/i)
    .click();
  });

  it('Delete a label successfully', () => {

    cy.contains('button', /delete/i)
    .click();

    // click again in pop up to confirm
    cy.contains('button', /delete/i)
    .click();

  });
});

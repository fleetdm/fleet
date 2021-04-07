describe('Query flow', () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
  });

  it('Create, check, edit, and delete a query successfully', () => {
    cy.visit('/queries/manage');

    cy.contains('button', /create new query/i).click();

    cy.get('.query-form__query-title').type('Query all window crashes');

    cy.get('.ace_content')
      .type('{selectall}{backspace}SELECT * FROM windows_crashes;')
      .click();

    cy.get('.query-form__query-description').type('See all window crashes');

    cy.contains('button', /save/i).click();

    cy.contains('button', /save as new/i).click();

    // no prompt to user that anything was done, just refreshes edit query

    cy.visit('/queries/manage');

    // see all window crashes in table
    cy.get('.queries-list-row__name').click();

    cy.contains('button', /edit or run query/i).click();

    cy.get('.ace_content')
      .type(
        '{selectall}{backspace}SELECT datetime, username FROM windows_crashes;'
      )
      .click();

    cy.contains('button', /save/i).click();

    cy.contains('button', /save changes/i).click();

    cy.get('.flash-message--success').should('be.visible');

    cy.visit('/queries/manage');

    cy.get('#query-checkbox-1').check({ force: true });

    cy.contains('button', /delete/i).click();

    cy.get('.manage-queries-page__modal-btn-wrap > .button--alert')
      .contains('button', /delete/i)
      .click();

    cy.get('.flash-message--success').should('be.visible');
  });
});

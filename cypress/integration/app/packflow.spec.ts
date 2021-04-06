// these run every hour for example

describe('Pack flow', () => {
    beforeEach(() => {
      cy.setup();
    });


    it('Create a pack successfully', () => {
        cy.visit('/packs/manage');

        cy.contains('button', /create new pack/i)
        .click();

        // query pack title

        // query pack description

        // dropdown select pack targets
        // have to hit the plus button...

        cy.contains('button', /save query pack/i)
        .click();

    });

    it('Edit a pack successfully', () => {
        cy.visit('/packs/manage');
        
        // click on query pack generated

        cy.contains('button', /edit pack/i)
        .click();

        // query pack title

        // query pack description

        // x all hosts
        // dropdown select pack targets
        // have to hit the plus button...
        // macos ?

        cy.contains('button', /save/i)
        .click();

    });

    it('Delete a pack successfully', () => {
        cy.visit('/packs/manage');

        // click on check box for the query pack generated

        cy.contains('button', /delete/i)
        .click();

        cy.contains('button', /delete/i)
        .click();
    });
});

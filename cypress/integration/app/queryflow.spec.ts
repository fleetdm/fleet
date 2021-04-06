// 


describe('Query flow', () => {
    beforeEach(() => {
      cy.setup();
    });

    it('Create a query successfully', () => {
        cy.visit('/queries/manage');

        cy.contains('button', /create new query/i)
        .click();

        // fill in query title
        // All window crashes

        // fill in SQL 
        // SELECT * FROM windows_crashes;

        // fill in description
        // See all info for window crashes

        cy.contains('button', /save/i)
        .click();

        // click dropdown save as new


        // this doesn't prompt the user that the query was saved
        // this doesn't redirect the user
        // it just refreshes the edit query
    });

    it('Check query shows up on query page', () => {
        cy.visit('/queries/manage');

        // see all window crashes in table

    });

    it('Edit a query successfully', () => {
        cy.visit('/queries/manage');

        // click on all windows crashes

        // click edit or run query

        // fill in query title

        // fill in SQL 

        // fill in description

        cy.contains('button', /save/i)
        .click();

        // click dropdown save changes

        // this prompts that it's successfully updated
    });

    it('Delete a query successfully', () => {
        cy.visit('/queries/manage');

        // click on check box next to all windows crashes

        cy.contains('button', /delete/i)
        .click();
    
        // click again in pop up to confirm
        cy.contains('button', /delete/i)
        .click();

    });
});
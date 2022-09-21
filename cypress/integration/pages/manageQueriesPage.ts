const manageQueriesPage = {
  visitManageQueriesPage: () => {
    cy.visit("/queries/manage");
  },

  hidesButton: (text: string) => {
    cy.contains("button", text).should("not.exist");
  },

  createsNewQuery: () => {
    cy.findByRole("button", { name: /new query/i }).click();
    cy.getAttached(".ace_scroller")
      .click({ force: true })
      .clear({ force: true })
      .type("{selectall}SELECT * FROM windows_crashes;", { force: true });
    cy.findByRole("button", { name: /save/i }).click();
    cy.getAttached(".modal__background").within(() => {
      cy.getAttached(".modal__modal_container").within(() => {
        cy.getAttached(".modal__content").within(() => {
          cy.getAttached("form").within(() => {
            cy.findByLabelText(/name/i).click().type("Cypress test query");
            cy.findByLabelText(/description/i)
              .click()
              .type("Cypress test of create new query flow.");
            cy.findByLabelText(/observers can run/i).click({ force: true });
            cy.findByRole("button", { name: /save query/i }).click();
          });
        });
      });
    });
    cy.findByText(/query created/i).should("exist");
    cy.getAttached(".query-form__query-name").within(() => {
      cy.findByText(/cypress test query/i).should("exist");
    });
  },

  allowsSelectTeamTargets: () => {
    cy.getAttached("tbody").within(() => {
      cy.getAttached("tr")
        .first()
        .within(() => {
          cy.getAttached(".fleet-checkbox__input").check({ force: true });
        });
      cy.findAllByText(/detect presence/i).click();
    });

    cy.getAttached(".query-form__button-wrap").within(() => {
      cy.findByRole("button", { name: /run/i }).click();
    });
    cy.contains("h3", /teams/i).should("exist");
    cy.contains(".selector-name", /apples/i).should("exist");
  },
};

export default manageQueriesPage;

import * as path from "path";
import { format } from "date-fns";

const manageQueriesPage = {
  visitManageQueriesPage: () => {
    cy.visit("/queries/manage");
  },

  hidesButton: (text: string) => {
    cy.contains("button", text).should("not.exist");
  },

  allowsCreateNewQuery: () => {
    cy.getAttached(".button--brand"); // ensures cta button loads
    cy.findByRole("button", { name: /new query/i }).click();
    cy.getAttached(".query-page__form .ace_scroller")
      .click({ force: true })
      .type("{selectall}SELECT * FROM windows_crashes;");
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
          });
        });
      });
    });
  },

  verifiesCreatedNewQuery: () => {
    cy.getAttached(".modal__background").within(() => {
      cy.getAttached(".modal__modal_container").within(() => {
        cy.getAttached(".modal__content").within(() => {
          cy.getAttached("form").within(() => {
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

  allowsEditExistingQuery: () => {
    cy.getAttached(".name__cell .button--text-link")
      .first()
      .click({ force: true });
    cy.getAttached(".query-page__form .ace_text-input")
      .click({ force: true })
      .clear({ force: true })
      .type("SELECT 1 FROM cypress;", {
        force: true,
      });
  },

  verifiesEditedExistingQuery: () => {
    cy.findByRole("button", { name: "Save" }).click(); // we have 'save as new' also
    cy.findByText(/query updated/i).should("be.visible");
  },

  allowsSaveAsNewQuery: () => {
    cy.getAttached(".name__cell .button--text-link")
      .eq(1)
      .within(() => {
        cy.findByText(/get authorized/i).click();
      });
    cy.findByRole("button", { name: /run query/i }).should("exist");
    cy.getAttached(".query-page__form .ace_scroller")
      .click()
      .type("{selectall}SELECT datetime, username FROM windows_crashes;");
    cy.findByRole("button", { name: /save as new/i }).should("be.enabled");
  },

  verifiesSavedAsNewQuery: () => {
    cy.findByRole("button", { name: /save as new/i }).click();
    cy.findByText(/successfully added query/i).should("be.visible");
    cy.findByText(/copy of/i).should("be.visible");
  },

  allowsDeleteExistingQuery: () => {
    cy.findByText(/detect presence of authorized ssh keys/i)
      .parent()
      .parent()
      .parent()
      .within(() => {
        cy.getAttached(".fleet-checkbox__input").check({
          force: true,
        });
      });
    cy.findByRole("button", { name: /delete/i }).click();
    cy.getAttached(".delete-query-modal .modal-cta-wrap").within(() => {
      cy.findByRole("button", { name: /delete/i }).should("exist");
    });
  },

  verifiesDeletedExistingQuery: () => {
    cy.getAttached(".delete-query-modal .modal-cta-wrap").within(() => {
      cy.findByRole("button", { name: /delete/i }).click();
    });
    cy.findByText(/successfully deleted query/i).should("be.visible");
    cy.findByText(/detect presence of authorized ssh keys/i).should(
      "not.exist"
    );
  },

  // TODO: Allows delete of self authored query only (Team Admin, team maintainer)

  allowsSelectTeamTargets: () => {
    cy.getAttached("tbody").within(() => {
      cy.findAllByText(/detect presence/i).click();
    });

    cy.getAttached(".query-form__button-wrap").within(() => {
      cy.findByRole("button", { name: /run/i }).click();
    });
    cy.contains("h3", /teams/i).should("exist");
    cy.contains(".selector-name", /apples/i).should("exist");
  },

  allowsRunQuery: () => {
    cy.getAttached(".name__cell .button--text-link").first().click();
    cy.findByRole("button", { name: /run query/i }).click();
    cy.findByText(/select targets/i).should("exist");
    cy.findByText(/all hosts/i).click();
    cy.findByText(/host targeted/i).should("exist"); // target count
  },

  verifiesRanQuery: () => {
    cy.findByRole("button", { name: /run/i }).click();
    cy.findByText(/querying selected host/i).should("exist"); // target count
  },

  allowsViewRanQuery: () => {
    // Ensures live query runs
    cy.wait(10000); // eslint-disable-line cypress/no-unnecessary-waiting
    cy.getAttached(".table-container").within(() => {
      cy.findByRole("button", { name: /show query/i }).click();
    });
    cy.getAttached(".show-query-modal").within(() => {
      cy.findByRole("button", { name: /done/i }).click();
    });
  },

  allowsExportQueryResults: () => {
    cy.getAttached(".table-container").within(() => {
      cy.findByRole("button", { name: /export results/i }).click();
      const formattedTime = format(new Date(), "MM-dd-yy hh-mm-ss");
      const filename = `Query Results (${formattedTime}).csv`;
      cy.readFile(path.join(Cypress.config("downloadsFolder"), filename), {
        timeout: 5000,
      });
    });
  },
};

export default manageQueriesPage;

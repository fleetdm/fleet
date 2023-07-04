const manageSchedulePage = {
  visitManageSchedulePage: () => {
    cy.visit("/schedule/manage");
  },

  hidesButton: (text: string) => {
    if (text === "Advanced") {
      cy.getAttached(".empty-table__cta-buttons").within(() => {
        cy.contains("button", text).should("not.exist");
      });
    } else cy.contains("button", text).should("not.exist");
  },

  changesTeam: (team1: string, team2: string) => {
    cy.getAttached(".manage-schedule-page__header").within(() => {
      cy.contains(team1).click({ force: true });
      cy.contains(team2).click({ force: true });
    });
  },

  confirmsTeam: (team: string) => {
    cy.getAttached(".manage-schedule-page__header-wrap").within(() => {
      cy.findByText(team).should("exist");
    });
  },

  allowsAddSchedule: () => {
    cy.getAttached(".empty-table__cta-buttons").within(() => {
      cy.findByRole("button", { name: /schedule a query/i }).click({
        force: true,
      });
    });
    cy.getAttached(".schedule-editor-modal__form").within(() => {
      cy.findByText(/select query/i).click();
      cy.findByText(/get local/i).click();
      cy.findByText(/every day/i).click();
      cy.findByText(/every 6 hours/i).click();
      cy.findByText(/show advanced options/i).click();
      cy.findByText(/snapshot/i).click();
      cy.findByText(/ignore removals/i).click();
      cy.getAttached(".schedule-editor-modal__form-field--platform").within(
        () => {
          cy.findByText(/select/i).click();
          cy.findByText(/linux/i).click();
        }
      );
      cy.getAttached(".schedule-editor-modal__form-field--osquer-vers").within(
        () => {
          cy.findByText(/all/i).click();
          cy.findByText(/4.6.0/i).click();
        }
      );
      cy.getAttached(".schedule-editor-modal__form-field--shard").within(() => {
        cy.getAttached(".input-field").click().type("50");
      });
      cy.getAttached(".modal-cta-wrap").within(() => {
        cy.findByRole("button", { name: /schedule/i }).should("be.enabled");
      });
    });
  },

  verifiesAddedSchedule: () => {
    cy.getAttached(".modal-cta-wrap").within(() => {
      cy.findByRole("button", { name: /schedule/i }).click();
    });
    cy.findByText(/successfully added/i).should("be.visible");
    cy.getAttached("tbody>tr").should("have.length", 1);
  },

  allowsEditSchedule: () => {
    cy.getAttached(".manage-schedule-page");
    cy.getAttached("tbody>tr")
      .should("have.length", 1)
      .within(() => {
        cy.findByText(/action/i).click();
        cy.findByText(/edit/i).click();
      });
    cy.getAttached(".schedule-editor-modal__form").within(() => {
      cy.findByText(/every 6 hours/i).click();
      cy.findByText(/every day/i).click();

      cy.getAttached(".modal-cta-wrap").within(() => {
        cy.findByRole("button", { name: /schedule/i }).should("be.enabled");
      });
    });
  },

  verifiesEditedSchedule: () => {
    cy.getAttached(".modal-cta-wrap").within(() => {
      cy.findByRole("button", { name: /schedule/i }).click();
    });
    cy.findByText(/successfully updated/i).should("be.visible");
  },

  allowsRemoveSchedule: () => {
    cy.getAttached(".manage-schedule-page");
    cy.getAttached("tbody>tr")
      .should("have.length", 1)
      .within(() => {
        cy.getAttached(".Select-placeholder").within(() => {
          cy.findByText(/action/i).click();
        });
        cy.getAttached(".Select-menu").within(() => {
          cy.findByText(/remove/i).click();
        });
      });
    cy.getAttached(".remove-scheduled-query-modal .modal-cta-wrap").within(
      () => {
        cy.findByRole("button", { name: /remove/i }).should("be.enabled");
      }
    );
  },

  verifiesRemovedSchedule: () => {
    cy.getAttached(".remove-scheduled-query-modal .modal-cta-wrap").within(
      () => {
        cy.findByRole("button", { name: /remove/i }).click();
      }
    );
    cy.findByText(/successfully removed/i).should("be.visible");
  },
};

export default manageSchedulePage;

const managePacksPage = {
  visitsManagePacksPage: () => {
    cy.visit("/packs/manage");
  },

  hidesButton: (text: string) => {
    cy.contains("button", text).should("not.exist");
  },

  allowsCreatePack: () => {
    cy.getAttached(".empty-table__container");
    cy.findByRole("button", { name: /create new pack/i }).click();
    cy.findByLabelText(/name/i).click().type("Errors and crashes");
    cy.findByLabelText(/description/i)
      .click()
      .type("See all user errors and window crashes.");
    cy.findByRole("button", { name: /save query pack/i }).should("be.enabled");
  },

  verifiesCreatedPack: () => {
    cy.findByRole("button", { name: /save query pack/i }).click();
  },

  allowsEditPack: () => {
    cy.findByLabelText(/name/i).clear().type("Server errors");
    cy.findByLabelText(/description/i)
      .clear()
      .type("See all server errors.");
    cy.findByRole("button", { name: /save/i }).should("be.enabled");
  },

  verifiesEditedPack: () => {
    cy.findByRole("button", { name: /save/i }).click();
  },

  allowsDeletePack: () => {
    cy.getAttached("tbody").within(() => {
      cy.getAttached("tr")
        .first()
        .within(() => {
          cy.getAttached(".fleet-checkbox__input").check({ force: true });
        });
    });
    cy.findByRole("button", { name: /delete/i }).click();
    cy.getAttached(".remove-pack-modal .modal-cta-wrap > .button--alert")
      .contains("button", /delete/i)
      .should("be.enabled");
  },

  verifiesDeletedPack: () => {
    cy.getAttached(".remove-pack-modal .modal-cta-wrap > .button--alert")
      .contains("button", /delete/i)
      .click({ force: true });
    cy.findByText(/successfully deleted/i).should("be.visible");
    managePacksPage.visitsManagePacksPage();
    cy.getAttached(".table-container").within(() => {
      cy.findByText(/windows starter pack/i).should("not.exist");
    });
  },
};

export default managePacksPage;

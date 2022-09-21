const manageSoftwarePage = {
  visitManageSoftwarePage: () => {
    cy.visit("/software/manage");
  },

  hidesButton: (text: string) => {
    if (text === "Manage automations") {
      cy.getAttached(".manage-software-page__header-wrap").within(() => {
        cy.contains("button", text).should("not.exist");
      });
    }
    cy.contains("button", text).should("not.exist");
  },

  allowsManageAutomations: () => {
    cy.findByRole("button", { name: /manage automations/i }).click();
    cy.findByRole("button", { name: /cancel/i }).click();
  },
};

export default manageSoftwarePage;

const manageSoftwarePage = {
  visitManageSoftwarePage: () => {
    cy.visit("/software/manage");
  },

  hidesButton: (text: string) => {
    cy.contains("button", text).should("not.exist");
  },

  allowsManageAutomations: () => {
    cy.findByRole("button", { name: /manage automations/i }).click();
    cy.findByRole("button", { name: /cancel/i }).click();
  },
};

export default manageSoftwarePage;

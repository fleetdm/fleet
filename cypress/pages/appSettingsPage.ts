const AppSettingsPage = {
  visitAgentOptions: () => {
    cy.visit("/settings/organization/agents");
  },

  editAgentOptionsForm: (text: string) => {
    cy.findByRole("textbox").type(text, { force: true });
    cy.findByRole("button", { name: /save/i }).should("be.enabled").click();
  },
};

export default AppSettingsPage;

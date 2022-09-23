const TeamSettingsPage = {
  visitTeamAgentOptions: (id: number) => {
    cy.visit(`settings/teams/${id}/options`);
  },

  editAgentOptionsForm: (text: string) => {
    cy.findByRole("textbox").type(text, { force: true });
    cy.findByRole("button", { name: /save/i }).click();
  },
};

export default TeamSettingsPage;

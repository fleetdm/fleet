const teamsDropdown = {
  switchTeams: (team1: string, team2: string) => {
    cy.getAttached(".component__team-dropdown").within(() => {
      cy.findByText(team1).click({ force: true });
    });
    cy.getAttached(".Select-menu").contains(team2).click({ force: true });
  },
};

export default teamsDropdown;

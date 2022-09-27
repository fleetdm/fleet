const userProfilePage = {
  visitUserProfilePage: () => {
    cy.visit("/profile");
  },

  showRole: (role: string, team?: string) => {
    cy.getAttached(".user-side-panel").within(() => {
      if (team) {
        cy.findByText(/team/i).next().contains(team);
      } else {
        cy.findByText(/teams/i).should("not.exist");
      }
      cy.findByText("Role").next().contains(role);
    });
  },
};

export default userProfilePage;

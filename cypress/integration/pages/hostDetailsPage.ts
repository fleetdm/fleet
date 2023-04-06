const hostDetailsPage = {
  visitsHostDetailsPage: (hostId: number) => {
    cy.visit(`/hosts/${hostId}`);
  },

  verifiesTeamsisDisabled: () => {
    cy.findByText(/team/i).should("not.exist");
  },

  verifiesTeam: (teamName: string) => {
    cy.getAttached(".info-flex").within(() => {
      // Team is shown for host
      cy.findByText(teamName).should("exist");
    });
  },

  hidesButton: (text: string) => {
    cy.contains("button", text).should("not.exist");
  },

  allowsCreateOsPolicy: () => {
    cy.getAttached(".info-flex").within(() => {
      cy.findByText(/ubuntu/i).should("exist");
      cy.getAttached(".host-summary__os-policy-button").click();
    });
    cy.getAttached(".modal__content")
      .findByRole("button", { name: /create new policy/i })
      .should("be.enabled");
  },

  hidesCreateOSPolicy: () => {
    cy.getAttached(".info-flex").within(() => {
      cy.findByText("Operating system")
        .next()
        .findByText(/ubuntu/i)
        .should("exist");

      cy.findByText("Operating system")
        .next()
        .findByRole("button")
        .should("not.exist");
    });
  },
};

export default hostDetailsPage;

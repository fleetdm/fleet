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

  deletesHost: () => {
    cy.findByRole("button", { name: /delete/i }).click();
    cy.findByText(/delete host/i).should("exist");
    cy.findByRole("button", { name: /cancel/i }).click();
  },

  transfersHost: () => {
    cy.getAttached(".host-details__transfer-button").click();
    cy.findByText(/create a team/i).should("exist");
    cy.getAttached(".Select-control").click();
    cy.getAttached(".Select-menu").within(() => {
      cy.findByText(/no team/i).should("exist");
      cy.findByText(/oranges/i).should("exist");
      cy.findByText(/apples/i).click();
    });
    cy.getAttached(".transfer-host-modal .modal-cta-wrap")
      .contains("button", /transfer/i)
      .click();
    cy.findByText(/transferred to apples/i).should("exist");
    cy.findByText(/team/i).next().contains("Apples");
  },

  queriesHost: () => {
    cy.findByRole("button", { name: /query/i }).click();
    cy.findByRole("button", { name: /create custom query/i }).should("exist");
    cy.getAttached(".modal__ex").within(() => {
      cy.findByRole("button").click();
    });
  },

  hidesCustomQuery: () => {
    cy.getAttached(".host-details__query-button").click();
    cy.contains("button", /create custom query/i).should("not.exist");
    cy.getAttached(".modal__ex").click();
  },

  createOperatingSystemPolicy: () => {
    cy.getAttached(".info-flex").within(() => {
      cy.findByText(/ubuntu/i).should("exist");
      cy.getAttached(".host-summary__os-policy-button").click();
    });
    cy.getAttached(".modal__content")
      .findByRole("button", { name: /create new policy/i })
      .should("exist");
  },

  hidesCreatingOSPolicy: () => {
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

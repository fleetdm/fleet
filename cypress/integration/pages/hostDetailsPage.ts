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

  allowsDeleteHost: () => {
    cy.findByRole("button", { name: /delete/i }).click();
    cy.getAttached(".modal__modal_container").within(() => {
      cy.findByRole("button", { name: /delete/i }).should("be.enabled");
    });
  },

  verifiesDeletedHost: (hostname: string) => {
    cy.getAttached(".modal__modal_container")
      .within(() => {
        cy.findByRole("button", { name: /delete/i }).click();
      })
      .then(() => {
        cy.findByText(/add your devices to fleet/i).should("exist");
        cy.findByText(/add hosts/i).should("exist");
        cy.findByText(/about this host/i).should("not.exist");
        cy.findByText(hostname).should("not.exist");
      });
  },

  allowsTransferHost: (create: boolean) => {
    cy.findByRole("button", { name: /transfer/i }).click();
    if (create) {
      cy.findByText(/create a team/i).should("exist");
    } else {
      cy.findByText(/create a team/i).should("not.exist");
    }
    cy.getAttached(".Select-control").click();
    cy.getAttached(".Select-menu").within(() => {
      cy.findByText(/no team/i).should("exist");
      cy.findByText(/oranges/i).should("exist");
      cy.findByText(/apples/i).click();
    });
    cy.getAttached(".transfer-host-modal .modal-cta-wrap")
      .contains("button", /transfer/i)
      .should("be.enabled");
  },

  verifiesTransferredHost: () => {
    cy.getAttached(".transfer-host-modal .modal-cta-wrap")
      .contains("button", /transfer/i)
      .click();
    cy.findByText(/transferred to apples/i).should("exist");
    cy.findByText(/team/i).next().contains("Apples");
  },

  allowsCustomQueryHost: () => {
    cy.findByRole("button", { name: /query/i }).click();
    cy.findByRole("button", { name: /create custom query/i }).should(
      "be.enabled"
    );
    cy.getAttached(".modal__ex").within(() => {
      cy.findByRole("button").click();
    });
  },

  hidesCustomQueryHost: () => {
    cy.findByRole("button", { name: /query/i }).click();
    cy.contains("button", /create custom query/i).should("not.exist");
    cy.getAttached(".modal__ex").click();
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

const manageHostsPage = {
  visitsManageHostsPage: () => {
    cy.visit("/hosts/manage");
  },

  allowsManageAndAddSecrets: () => {
    cy.contains("button", /manage enroll secret/i).click();
    cy.contains("button", /add secret/i).click();
    cy.contains("button", /save/i).click();
    cy.contains("button", /done/i).click();
  },

  allowsAddHosts: () => {
    cy.getAttached(".team-details__action-buttons--primary")
      .contains("button", /add hosts/i)
      .click();
    cy.getAttached(".modal__content").contains("button", /done/i).click();
  },

  allowsAddLabelForm: () => {
    cy.getAttached(".label-filter-select__control").click();
    cy.findByRole("button", { name: /add label/i }).click();
    cy.findByText(/New Label/i).should("exist");
  },

  hidesButton: (text: string) => {
    if (text === "Add label") {
      cy.getAttached(".label-filter-select__control").click();
      cy.contains("button", /add label/i).should("not.exist");
    } else {
      cy.contains("button", text).should("not.exist");
    }
  },

  includesTeamColumn: () => {
    cy.getAttached("thead").within(() => {
      cy.findByText(/team/i).should("exist");
    });
  },

  ensuresTeamDropdownLoads: (teamName = "All teams") => {
    cy.getAttached(".Select-value-label").contains(teamName);
  },

  verifiesTeamsIsDisabled: () => {
    cy.findByText(/teams/i).should("not.exist");
  },
};

export default manageHostsPage;

import TeamSettingsPage from "../../pages/teamSettingsPage";

describe("Dashboard", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
  });

  after(() => {
    cy.logout();
  });

  describe("Activity Card", () => {
    it("displays activity when editing a teams agent options", () => {
      cy.visit("/settings/teams");

      cy.getAttached(".no-teams__inner-text").within(() => {
        cy.findByRole("button", { name: "Create team" })
          .should("be.visible")
          .click();
      });

      cy.findByRole("textbox", { name: "Team name" }).type("Team 1");
      cy.findByRole("button", { name: "Create" }).click();

      // we've just created this team so we know we only have one team with id = 1
      TeamSettingsPage.visitTeamAgentOptions(1);
      TeamSettingsPage.editAgentOptionsForm("test:");

      cy.visit("/dashboard");

      // the edit agent options is split across multiple elements so we use a
      // matcher function and assert the different parts individually.
      cy.findByText((content) => content.includes("edited agent options on"))
        .should("exist")
        .and("contain", "Admin")
        .and("contain", "Team 1")
        .and("contain", "team");
    });
  });
});

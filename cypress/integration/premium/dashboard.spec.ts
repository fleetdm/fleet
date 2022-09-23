// import TeamSettingsPage from "../../pages/teamSettingsPage";

// describe("Dashboard", () => {
//   before(() => {
//     Cypress.session.clearAllSavedSessions();
//     cy.setup();
//     cy.loginWithCySession();
//     cy.viewport(1200, 660);
//   });

//   after(() => {
//     cy.logout();
//   });

//   describe("Activity Card", () => {
//     beforeEach(() => {
//       cy.loginWithCySession();
//     });

//     it("displays activity when editing a teams agent options", () => {
//       cy.visit("/settings/teams");
//       cy.intercept("GET", "/api/latest/fleet/activities?*").as("getActivities");

//       cy.getAttached(".no-teams").within(() => {
//         cy.getAttached(".no-teams__inner-text").within(() => {
//           cy.contains("button", /create team/i).click();
//         });
//       });

//       cy.findByRole("textbox", { name: "Team name" }).type("Team 1");
//       cy.findByRole("button", { name: "Create" }).click();

//       // we've just created this team so we know we only have one team with id = 1
//       TeamSettingsPage.visitTeamAgentOptions(1);

//       cy.intercept("GET", "/api/latest/fleet/teams?*").as("getTeams");

//       cy.wait("@getTeams").then(() => {
//         TeamSettingsPage.editAgentOptionsForm(
//           "{selectall}{backspace}test: null{enter}"
//         );
//         cy.getAttached(".flash-message").should("exist");
//       });

//       cy.visit("/dashboard");

//       cy.wait("@getActivities").then(() => {
//         // the edit agent options is split across multiple elements so we use a
//         // matcher function and assert the different parts individually.
//         cy.getAttached(".activity-feed__block").within(() => {
//           cy.getAttached(".activity-feed__details-topline")
//             .first()
//             .contains(/edited agent options on/gi)
//             .contains(/Admin/gi)
//             .contains(/Team 1/gi);
//         });
//       });
//     });
//   });
// });

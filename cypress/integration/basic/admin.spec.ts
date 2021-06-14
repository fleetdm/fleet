/* eslint-disable cypress/no-unnecessary-waiting */
if (Cypress.env("FLEET_TIER") === "basic") {
  describe("Basic tier - Admin user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedBasic();
      cy.setupSMTP();
      cy.logout();
    });

    it("Can perform the appropriate basic-tier admin actions", () => {
      cy.login("anna@organization.com", "user123#");
      cy.visit("/");

      // Ensure the hosts page is loaded
      cy.contains("All hosts");

      // On the hosts page, they should…

      // See the “Teams” column in the Hosts table
      // cy.findByRole("columnheader", { name: "Team" });
      // ^^TODO this test depends on seeding hosts because the table is not displayed if there are no hosts

      // See and select the “Add new host” button
      cy.contains("button", /add new host/i).click();

      // See the “Select a team for this new host” in the Add new host modal. This modal appears after the user selects the “Add new host” button
      cy.wait(1000);
      cy.get(".Select-control").click();
      cy.wait(1000);
      cy.findByText(/no team/i).should("exist");
      // cy.findByText(/apples/i).should("exist");
      // cy.findByText(/oranges/i).should("exist");
      // ^^TODO investigate why this dropdown is not being populated with the appropriate teams (i.e. Apples and Oranges) when Cypress attempts this test; note that issue only occurs during the automated test run and the dropdown is populated correctly if you comment out the assertions for apples and organges and manually navigate to the modal after the test run is finished

      cy.contains("button", /done/i).click();

      // On the Host details page, they should…
      // See the “Team” information below the hostname
      // cy.visit("/hosts/2");
      // cy.findByText(/team/i).next().contains("Apples");
      // ^^TODO this test depends on seeding hosts

      // On the Queries - new / edit / run page, they should…
      // See the “Teams” section in the Select target picker. This picker is summoned when the “Select targets” field is selected.
      cy.visit("/queries/new");
      cy.get(".target-select").within(() => {
        cy.findByText(/Label name, host name, IP address, etc./i).click();
        cy.findByText(/teams/i).should("exist");
      });

      // On the Packs pages (manage, new, and edit), they should…
      // ^^TODO

      // On the Settings pages, they should…
      // See the “Teams” navigation item
      cy.visit("/settings/organization");
      cy.get(".react-tabs").within(() => {
        cy.findByText(/teams/i).click();
      });
      cy.findByText(/apples/i).click();
      cy.findByText(/Add and remove members from Apples/i).should("exist");

      cy.visit("/settings/organization");
      cy.findByText(/teams/i).should("exist");

      // Access the Settings - Teams page
      // ^^ TODO

      // Access the Settings - Team details page
      // ^^ TODO

      // See the “Team” section in the create user modal. This modal is summoned when the “Create user” button is selected
      cy.get(".react-tabs").within(() => {
        cy.findByText(/users/i).click();
      });
      cy.findByRole("button", { name: /create user/i }).click();
      // cy.findByText(/team/i).should("exist");
      // ^^TODO add this test after bug is fixed
    });
  });
}

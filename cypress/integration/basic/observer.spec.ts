if (Cypress.env("FLEET_TIER") === "basic") {
  describe("Basic tier - Observer user", () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedBasic();
      cy.seedQueries([
        {name: "Detect presence of authorized SSH keys", query: "SELECT username, authorized_keys. * FROM users CROSS JOIN authorized_keys USING (uid)", description: "Presence of authorized SSH keys may be unusual on laptops. Could be completely normal on servers, but may be worth auditing for unusual keys and/or changes.", observer_can_run: true},
        { name: "Get authorized keys for Domain Joined Accounts", query: "SELECT * FROM users CROSS JOIN authorized_keys USING(uid) WHERE username IN (SELECT distinct(username) FROM last);", description: "List authorized_keys for each user on the system.", observer_can_run: false },
        { name: "Get crashes", query: "SELECT uid, datetime, responsible, exception_type, identifier, version, crash_path FROM users CROSS JOIN crashes USING (uid);", description: "Retrieve application, system, and mobile app crash logs.", observer_can_run: false },
        { name: "Get installed Chrome Extensions", query: "SELECT uid, datetime, responsible, exception_type, identifier, version, crash_path FROM users CROSS JOIN crashes USING (uid);", description: "List installed Chrome Extensions for all users.", observer_can_run: false },
        { name: "Get installed Safari extensions", query: "SELECT safari_extensions.* FROM users join safari_extensions USING (uid);", description: "Retrieves the list of installed Safari Extensions for all users in the target system.", observer_can_run: false },
      ]);
      cy.logout();
    });

    it("Can perform the appropriate actions", () => {
      cy.login("oliver@organization.com", "user123#");
      cy.visit("/");

      // Ensure page is loaded
      cy.contains("All hosts");

      // TODO write the test!
    });

    it("Should verify Teams on Hosts page", () => {
      cy.login("marco@organization.com", "user123#");
      cy.visit("/hosts/manage");
  
      cy.findByText("All hosts which have enrolled in Fleet").should("exist");

      // TODO: can see the "Team" column in the Hosts table
      // cy.contains(".table-container .data-table__table th", "Team").should("be.visible");
    });

    it("Should verify hidden items on Hosts page", () => {
      cy.login("marco@organization.com", "user123#");
      cy.visit("/hosts/manage");
  
      cy.findByText("Packs").should("not.exist");
      cy.findByText("Packs").should("not.exist");

      // TODO: can see the "Team" column in the Hosts table
      // cy.contains(".table-container .data-table__table th", "Team").should("be.visible");
    });
  });
}

// describe(
//   "Policies flow",
//   {
//     defaultCommandTimeout: 20000,
//   },
//   () => {
//     beforeEach(() => {
//       cy.setup();
//       cy.login();
//       cy.seedQueries();
//     });
//     it("Can create, check, and delete a policy successfully", () => {
//       cy.intercept({
//         method: "GET",
//         url: "/api/v1/fleet/global/policies",
//       }).as("getPolicies");
//       cy.intercept({
//         method: "GET",
//         url: "/api/v1/fleet/config",
//       }).as("getConfig");

//       cy.visit("/policies/manage");

//       // wait for state of policy table to settle otherwise re-renders cause elements to detach and tests will fail
//       cy.wait("@getPolicies");
//       cy.wait("@getConfig");

//       cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting

//       // Add a policy
//       cy.get(".no-policies__inner")
//         .findByText(/add a policy/i)
//         .should("exist")
//         .click();
//       cy.get(".add-policy-modal").within(() => {
//         cy.findByText(/select query/i)
//           .should("exist")
//           .click();
//         cy.findByText(
//           /Detect Linux hosts with high severity vulnerable versions of OpenSSL/i
//         ).click();
//         cy.findByRole("button", { name: /cancel/i }).should("exist");
//         cy.findByRole("button", { name: /add/i }).should("exist").click();
//       });

//       // Confirm that policy was added successfully
//       cy.findByText(/successfully added policy/i).should("exist");
//       cy.findByText(/select query/i).should("not.exist");
//       cy.get(".policies-list-wrapper").within(() => {
//         cy.findByText(/1 query/i).should("exist");
//         cy.findByText(/yes/i).should("exist");
//         cy.findByText(
//           /Detect Linux hosts with high severity vulnerable versions of OpenSSL/i
//         ).should("exist");

//         // Click on link in table and confirm that policies filter block diplays as expected on manage hosts page
//         cy.get("tbody").within(() => {
//           cy.get("tr")
//             .first()
//             .within(() => {
//               cy.get("td").last().children().first().should("exist").click();
//             });
//         });
//       });
//       cy.get(".manage-hosts__policies-filter-block").within(() => {
//         cy.findByText(
//           /Detect Linux hosts with high severity vulnerable versions of OpenSSL/i
//         ).should("exist");
//         cy.findByText(/yes/i).should("not.exist");
//         cy.findByText(/failing/i)
//           .should("exist")
//           .click();
//         cy.findByText(/yes/i).should("exist");
//         cy.get('img[alt="Remove policy filter"]').click();
//         cy.findByText(
//           /Detect Linux hosts with high severity vulnerable versions of OpenSSL/i
//         ).should("not.exist");
//       });

//       // Click on policies tab to return to manage policies page
//       cy.get(".site-nav-container").within(() => {
//         cy.findByText(/policies/i)
//           .should("exist")
//           .click();
//       });

//       // Delete policy
//       cy.get("tbody").within(() => {
//         cy.get("tr")
//           .first()
//           .within(() => {
//             cy.get(".fleet-checkbox__input").check({ force: true });
//           });
//       });
//       cy.findByRole("button", { name: /remove/i }).click();
//       cy.get(".remove-policies-modal").within(() => {
//         cy.findByRole("button", { name: /cancel/i }).should("exist");
//         cy.findByRole("button", { name: /remove/i })
//           .should("exist")
//           .click();
//       });
//       cy.findByText(
//         /Detect Linux hosts with high severity vulnerable versions of OpenSSL/i
//       ).should("not.exist");
//     });
//   }
// );

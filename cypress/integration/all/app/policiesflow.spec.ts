describe(
  "Policies flow",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
    });
    it("Can create, update, delete a policy successfully, and turn on failing policies webhook", () => {
      cy.visit("/policies/manage");
      cy.get(".manage-policies-page__description")
        .should("contain", /add policies/i)
        .and("contain", /manage automations/i); // Ensure page load

      // Add a policy
      cy.findByText(/add a policy/i).click();
      cy.findByText(/create your own policy/i).click();

      cy.get(".ace_scroller")
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM users WHERE username = 'backup' LIMIT 1;"
        );

      cy.findByRole("button", { name: /save policy/i }).click();

      // save modal
      cy.get(".policy-form__policy-save-modal-name")
        .click()
        .type("Does the device have a user named 'backup'?");

      cy.get(".policy-form__policy-save-modal-description")
        .click()
        .type("Returns yes or no for having a user named 'backup'");

      cy.get(".policy-form__policy-save-modal-resolution")
        .click()
        .type("Create a user named 'backup'");

      cy.findByRole("button", { name: /^Save$/ }).click();

      // Confirm that policy was added successfully
      cy.findByText(/policy created/i).should("exist");

      cy.visit("/policies/manage");
      cy.get(".manage-policies-page__description").should(
        "contain",
        /add policies/i
      ); // Ensure page load

      // Add a default policy
      cy.findByText(/add a policy/i).click();

      cy.findByText(/gatekeeper enabled/i).click();
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.findByRole("button", { name: /^Save$/ }).click();

      // Confirm that policy was added successfully
      cy.findByText(/policy created/i).should("exist");

      cy.visit("/policies/manage");

      // Click on link in table and confirm that policies filter block diplays as expected on manage hosts page
      cy.getAttached(".failing_host_count__cell")
        .first()
        .within(() => {
          cy.findByRole("button", { name: /0 hosts/i }).click();
        });

      // confirm policy functionality on manage host page
      cy.get(".manage-hosts__policies-filter-block").within(() => {
        cy.findByText(/user named 'backup'/i).should("exist");
        cy.findByText(/no/i).should("exist").click();
        cy.findByText(/yes/i).should("exist");
        cy.get('img[alt="Remove policy filter"]').click();
        cy.findByText(/user named 'backup'/i).should("not.exist");
      });

      // Click on policies tab to return to manage policies page
      cy.get(".site-nav-container").within(() => {
        cy.findByText(/policies/i).click();
      });

      // Update policy
      cy.findByText(/gatekeeper enabled/i).click();

      cy.get(".ace_scroller")
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;"
        );

      cy.findByRole("button", { name: /^Save$/ }).click();

      // Confirm that policy was added successfully
      cy.findByText(/policy updated/i).should("exist");

      // Delete policy
      cy.visit("/policies/manage");

      cy.getAttached("tbody").within(() => {
        cy.get("tr")
          .first()
          .within(() => {
            cy.get(".fleet-checkbox__input").check({ force: true });
          });
      });
      cy.findByRole("button", { name: /delete/i }).click();
      cy.get(".remove-policies-modal").within(() => {
        cy.findByRole("button", { name: /cancel/i }).should("exist");
        cy.findByRole("button", { name: /delete/i }).click();
      });
      cy.findByText(/removed policy/i).should("exist");
      cy.findByText(/backup/i).should("not.exist");

      // Create failing policies webhook
      cy.findByRole("button", { name: /manage automations/i }).click();
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.get(".fleet-checkbox__input").check({ force: true });
      });
      cy.get("#webhook-url").click().type("www.foo.com/bar");
      cy.findByRole("button", { name: /^Save$/ }).click();

      // Confirm that failing policies webhook was added successfully
      cy.findByText(/updated policy automations/i).should("exist");
      cy.findByRole("button", { name: /manage automations/i }).click();
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.get(".fleet-checkbox__input").should("be.checked");
      });
    });
  }
);

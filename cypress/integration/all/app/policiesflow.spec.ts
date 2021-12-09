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
    it("Can create, update, and delete a policy successfully", () => {
      cy.visit("/policies/manage");
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting

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
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting

      // Add a default policy
      cy.findByText(/add a policy/i).click();

      cy.findByText(/gatekeeper enabled/i).click();
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.findByRole("button", { name: /^Save$/ }).click();

      // Confirm that policy was added successfully
      cy.findByText(/policy created/i).should("exist");

      cy.visit("/policies/manage");
      cy.wait(2000); // eslint-disable-line cypress/no-unnecessary-waiting

      cy.get(".policies-list-wrapper").within(() => {
        cy.findByText(/backup/i).should("exist");
        cy.findByText(/gatekeeper/i).should("exist");

        // Click on link in table and confirm that policies filter block diplays as expected on manage hosts page
        cy.get("tbody").within(() => {
          cy.get("tr")
            .first()
            .within(() => {
              cy.get("td").last().children().first().should("exist").click();
            });
        });
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
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting

      // TODO: Detached dom issues
      // cy.findByText(/gatekeeper enabled on macOS/i)
      //   .click()
      //   .type("{selectall}Gatekeeper enabled on macOS");
      // cy.findByText(/checks to make sure/i)
      //   .click()
      //   .type(
      //     "{selectall}Gatekeeper helps ensure only trusted software is running"
      //   );
      // cy.findByText(/failing device/i)
      //   .click()
      //   .type("{selectall}Run /user/sbin/spctl--master -enable");

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
      cy.wait(2000); // eslint-disable-line cypress/no-unnecessary-waiting

      cy.get("tbody").within(() => {
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
    });
  }
);

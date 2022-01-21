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
    it("Create, update, and delete a policy successfully; turn on failing policies webhook", () => {
      cy.visit("/policies/manage");

      // Add a policy
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });

      cy.findByText(/create your own policy/i).click();

      cy.getAttached(".ace_scroller")
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM users WHERE username = 'backup' LIMIT 1;"
        );

      cy.findByRole("button", { name: /save policy/i }).click();

      // save modal
      cy.getAttached(".policy-form__policy-save-modal-name")
        .click()
        .type("Does the device have a user named 'backup'?");

      cy.getAttached(".policy-form__policy-save-modal-description")
        .click()
        .type("Returns yes or no for having a user named 'backup'");

      cy.getAttached(".policy-form__policy-save-modal-resolution")
        .click()
        .type("Create a user named 'backup'");

      cy.findByRole("button", { name: /^Save$/ }).click();

      cy.findByText(/policy created/i).should("exist");

      // Add a default policy
      cy.visit("/policies/manage");
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });

      cy.findByText(/gatekeeper enabled/i).click();
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.findByRole("button", { name: /^Save$/ }).click();

      cy.findByText(/policy created/i).should("exist");

      cy.visit("/policies/manage");

      // Policy filter diplays on manage hosts page
      cy.getAttached(".failing_host_count__cell")
        .first()
        .within(() => {
          cy.getAttached(".button--text-link").click();
        });

      // Confirm policy functionality on manage host page
      cy.getAttached(".manage-hosts__policies-filter-block").within(() => {
        cy.findByText(/user named 'backup'/i).should("exist");
        cy.findByText(/no/i).should("exist").click();
        cy.findByText(/yes/i).should("exist");
        cy.get('img[alt="Remove policy filter"]').click();
        cy.findByText(/user named 'backup'/i).should("not.exist");
      });

      // Update policy
      cy.visit("/policies/manage");
      cy.getAttached(".name__cell .button--text-link").last().click();

      cy.getAttached(".ace_scroller")
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;"
        );

      cy.getAttached(".policy-form__save").click();

      cy.findByText(/policy updated/i).should("exist");

      // Delete policy
      cy.visit("/policies/manage");

      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.getAttached(".fleet-checkbox__input").check({ force: true });
          });
      });
      cy.findByRole("button", { name: /delete/i }).click();
      cy.getAttached(".remove-policies-modal").within(() => {
        cy.findByRole("button", { name: /cancel/i }).should("exist");
        cy.findByRole("button", { name: /delete/i }).click();
      });
      cy.findByText(/removed policy/i).should("exist");
      cy.findByText(/backup/i).should("not.exist");

      // Create failing policies webhook
      cy.findByRole("button", { name: /manage automations/i }).click();
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-checkbox__input").check({ force: true });
      });
      cy.getAttached("#webhook-url").click().type("www.foo.com/bar");
      cy.findByRole("button", { name: /^Save$/ }).click();

      // Confirm failing policies webhook was added successfully
      cy.findByText(/updated policy automations/i).should("exist");
      cy.findByRole("button", { name: /manage automations/i }).click();
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-checkbox__input").should("be.checked");
      });
    });
  }
);

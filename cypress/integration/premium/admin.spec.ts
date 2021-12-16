describe(
  "Premium tier - Admin user",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    beforeEach(() => {
      cy.setup();
      cy.login();
      cy.seedPremium();
      cy.setupSMTP();
      cy.seedQueries();
      cy.addDockerHost("apples");
      cy.logout();
    });
    afterEach(() => {
      cy.stopDockerHost();
    });

    it(
      "Can perform the appropriate premium-tier admin actions",
      {
        retries: {
          runMode: 2,
        },
      },
      () => {
        cy.login("anna@organization.com", "user123#");
        cy.visit("/hosts/manage");

        // Ensure the hosts page is loaded
        cy.contains("All hosts");

        // On the hosts page, they should…

        // See the “Teams” column in the Hosts table
        cy.get("thead").contains(/team/i).should("exist");

        // See and select the “Generate installer” button
        cy.contains("button", /generate installer/i).click();
        cy.contains("button", /done/i).click();

        // See the "Manage" enroll secret” button. A modal appears after the user selects the button
        // Add secret tests same API as edit and delete
        cy.contains("button", /manage enroll secret/i).click();
        cy.contains("button", /add secret/i).click();
        cy.contains("button", /save/i).click();
        cy.contains("button", /done/i).click();

        // On the Host details page, they should…
        // See the “Team” information below the hostname
        // Be able to transfer Teams
        cy.visit("/hosts/1");
        cy.findByText(/team/i).next().contains("Apples");
        cy.contains("button", /transfer/i).click();
        cy.get(".Select-control").click();
        cy.findByText(/create a team/i).should("exist");
        cy.get(".Select-menu").within(() => {
          cy.findByText(/no team/i).should("exist");
          cy.findByText(/apples/i).should("exist");
          cy.findByText(/oranges/i).click();
        });
        cy.get(".transfer-action-btn").click();
        cy.findByText(/transferred to oranges/i).should("exist");
        cy.findByText(/team/i).next().contains("Oranges");
        // See and select operating system
        // TODO

        // TODO - Fix tests according to improved query experience - MP
        // On the Queries - new / edit / run page, they should…
        // See the “Teams” section in the Select target picker. This picker is summoned when the “Select targets” field is selected.
        // cy.visit("/queries/new");
        // cy.get(".target-select").within(() => {
        //   cy.findByText(/Label name, host name, IP address, etc./i).click();
        //   cy.findByText(/teams/i).should("exist");
        // });

        cy.visit("/queries/manage");

        cy.findByRole("button", { name: /create new query/i }).click();

        // Using class selector because third party element doesn't work with Cypress Testing Selector Library
        cy.get(".ace_scroller")
          .click({ force: true })
          .type("{selectall}SELECT * FROM windows_crashes;");

        cy.findByRole("button", { name: /save/i }).click();

        // save modal
        cy.get(".query-form__query-save-modal-name")
          .click()
          .type("Query all window crashes");

        cy.get(".query-form__query-save-modal-description")
          .click()
          .type("See all window crashes");

        cy.findByRole("button", { name: /save query/i }).click();

        cy.findByText(/query created/i).should("exist");
        cy.findByText(/back to queries/i).should("exist");
        cy.visit("/queries/manage");

        cy.wait(2000); // eslint-disable-line cypress/no-unnecessary-waiting
        cy.findByText(/query all/i).click();

        cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
        cy.findByText(/run query/i).should("exist");

        cy.get(".ace_scroller")
          .click({ force: true })
          .type("{selectall}SELECT datetime, username FROM windows_crashes;");

        cy.findByRole("button", { name: /^Save$/ }).click();

        cy.findByText(/query updated/i).should("be.visible");

        // Start e2e test for schedules
        cy.visit("/schedule/manage");

        cy.wait(2000); // eslint-disable-line cypress/no-unnecessary-waiting

        cy.findByRole("button", { name: /schedule a query/i }).click();

        cy.findByText(/select query/i).click();

        cy.findByText(/query all window crashes/i).click();

        cy.get(
          ".schedule-editor-modal__form-field--frequency > .dropdown__select"
        ).click();

        cy.findByText(/every week/i).click();

        cy.findByText(/show advanced options/i).click();

        cy.get(
          ".schedule-editor-modal__form-field--logging > .dropdown__select"
        ).click();

        cy.findByText(/ignore removals/i).click();

        cy.get(".schedule-editor-modal__form-field--shard > .input-field")
          .click()
          .type("50");

        cy.get(".schedule-editor-modal__btn-wrap")
          .contains("button", /schedule/i)
          .click();

        cy.visit("/schedule/manage");

        cy.wait(2000); // eslint-disable-line cypress/no-unnecessary-waiting
        cy.findByText(/query all window crashes/i).should("exist");

        cy.findByText(/actions/i).click();
        cy.findByText(/edit/i).click();

        cy.get(
          ".schedule-editor-modal__form-field--frequency > .dropdown__select"
        ).click();

        cy.findByText(/every 6 hours/i).click();

        cy.findByText(/show advanced options/i).click();

        cy.findByText(/ignore removals/i).click();
        cy.findByText(/snapshot/i).click();

        cy.get(".schedule-editor-modal__form-field--shard > .input-field")
          .click()
          .type("{selectall}{backspace}10");

        cy.get(".schedule-editor-modal__btn-wrap")
          .contains("button", /schedule/i)
          .click();

        cy.visit("/schedule/manage");

        cy.wait(2000); // eslint-disable-line cypress/no-unnecessary-waiting
        cy.findByText(/actions/i).click();
        cy.findByText(/remove/i).click();

        cy.get(".remove-scheduled-query-modal__btn-wrap")
          .contains("button", /remove/i)
          .click();

        cy.findByText(/query all window crashes/i).should("not.exist");

        // End e2e test for schedules

        cy.visit("/queries/manage");
        cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
        cy.findByText(/query all window crashes/i)
          .parent()
          .parent()
          .within(() => {
            cy.get(".fleet-checkbox__input").check({ force: true });
          });

        cy.findByRole("button", { name: /delete/i }).click();

        // Can't figure out how attach findByRole onto modal button
        // Can't use findByText because delete button under modal
        cy.get(".remove-query-modal")
          .contains("button", /delete/i)
          .click();

        cy.findByText(/successfully removed query/i).should("be.visible");

        cy.findByText(/query all/i).should("not.exist");

        // On the Packs pages (manage, new, and edit), they should…
        // ^^General admin functionality for packs page is being tested in app/packflow.spec.ts

        // On the Settings pages, they should…
        // See the “Teams” navigation item and access the Settings - Teams page
        cy.visit("/settings/organization");
        cy.get(".react-tabs").within(() => {
          cy.findByText(/teams/i).click();
        });
        // Access the Settings - Team details page
        cy.findByText(/apples/i).click();
        cy.findByText(/apples/i).should("exist");
        cy.findByText(/manage users with global access here/i).should("exist");

        // See the “Team” section in the create user modal. This modal is summoned when the “Create user” button is selected
        cy.visit("/settings/organization");
        cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
        cy.get(".react-tabs").within(() => {
          cy.findByText(/users/i).click();
        });
        cy.findByRole("button", { name: /create user/i }).click();
        cy.findByText(/assign teams/i).should("exist");

        // On the Profile page, they should…
        // See Global in the Team section and Admin in the Role section
        cy.visit("/profile");
        cy.findByText(/team/i)
          .next()
          .contains(/global/i);
        cy.findByText("Role").next().contains(/admin/i);
      }
    );
  }
);

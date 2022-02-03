describe("Premium tier - Admin user", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedPremium();
    cy.seedQueries();
    cy.seedPolicies("apples");
    cy.addDockerHost("apples");
  });
  after(() => {
    cy.logout();
    cy.stopDockerHost();
  });

  describe("Global admin", () => {
    beforeEach(() =>
      cy.loginWithCySession("anna@organization.com", "user123#")
    );
    describe("Dashboard and navigation", () => {
      beforeEach(() => cy.visit("/dashboard"));
      it("displays intended global admin dashboard", () => {
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/all teams/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".hosts-status").should("exist");
          cy.getAttached(".home-software").should("exist");
          cy.getAttached(".activity-feed").should("exist");
        });
      });
      it("displays intended global admin top navigation", () => {
        cy.getAttached(".site-nav-container").within(() => {
          cy.findByText(/hosts/i).should("exist");
          cy.findByText(/software/i).should("exist");
          cy.findByText(/queries/i).should("exist");
          cy.findByText(/schedule/i).should("exist");
          cy.findByText(/policies/i).should("exist");
          cy.getAttached(".user-menu").click();
          cy.findByText(/settings/i).click();
        });
        cy.getAttached(".react-tabs__tab--selected").within(() => {
          cy.findByText(/organization/i).should("exist");
        });
        cy.getAttached(".site-nav-container").within(() => {
          cy.getAttached(".user-menu").click();
          cy.findByText(/manage users/i).click();
        });
        cy.getAttached(".react-tabs__tab--selected").within(() => {
          cy.findByText(/users/i).should("exist");
        });
      });
    });
    describe("Manage hosts page", () => {
      beforeEach(() => cy.visit("/hosts/manage"));
      it("displays team column in hosts table", () => {
        cy.getAttached(".data-table__table th")
          .contains("Team")
          .should("be.visible");
      });
      it("allows global admin to see and click generate installer", () => {
        cy.getAttached(".button-wrap")
          .contains("button", /generate installer/i)
          .click();
        cy.getAttached(".modal__content").contains("button", /done/i).click();
      });
      it("allows global admin to add new enroll secret", () => {
        cy.getAttached(".button-wrap")
          .contains("button", /manage enroll secret/i)
          .click();
        cy.getAttached(".enroll-secret-modal__add-secret")
          .contains("button", /add secret/i)
          .click();
        cy.getAttached(".secret-editor-modal__button-wrap")
          .contains("button", /save/i)
          .click();
        cy.getAttached(".enroll-secret-modal__button-wrap")
          .contains("button", /done/i)
          .click();
      });
    });
    describe("Host details page", () => {
      beforeEach(() => cy.visit("hosts/1"));
      it("allows global admin to transfer host to an existing team", () => {
        cy.getAttached(".host-details__transfer-button").click();
        cy.findByText(/create a team/i).should("exist");
        cy.getAttached(".Select-control").click();
        cy.getAttached(".Select-menu").within(() => {
          cy.findByText(/no team/i).should("exist");
          cy.findByText(/apples/i).should("exist");
          cy.findByText(/oranges/i).click();
        });
        cy.getAttached(".transfer-action-btn").click();
        cy.findByText(/transferred to oranges/i).should("exist");
        cy.findByText(/team/i).next().contains("Oranges");
      });
      it("allows global admin to create an operating system policy", () => {
        cy.getAttached(".info-flex").within(() => {
          cy.findByText(/ubuntu/i).should("exist");
          cy.getAttached(".host-details__os-policy-button").click();
        });
        cy.getAttached(".modal__content")
          .findByRole("button", { name: /create new policy/i })
          .should("exist");
      });
      it("allows global admin to create a custom query", () => {
        cy.getAttached(".host-details__query-button").click();
        cy.contains("button", /create custom query/i).should("exist");
        cy.getAttached(".modal__ex").click();
      });
      it("allows global admin to delete a host", () => {
        cy.getAttached(".host-details__action-button-container")
          .contains("button", /delete/i)
          .click();
        cy.getAttached(".host-details__modal").within(() => {
          cy.findByText(/delete host/i).should("exist");
          cy.contains("button", /delete/i).should("exist");
          cy.getAttached(".modal__ex").click();
        });
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => cy.visit("/software/manage"));
      it("allows global admin to click 'Manage automations' butto", () => {
        cy.getAttached(".manage-software-page__header-wrap").within(() => {
          cy.getAttached(".Select").within(() => {
            cy.findByText(/all teams/i).should("exist");
          });
          cy.findByRole("button", { name: /manage automations/i }).click();
          cy.findByRole("button", { name: /cancel/i }).click();
        });
      });
      it("hides manage automations button when all teams not selected", () => {
        cy.getAttached(".manage-software-page__header-wrap").within(() => {
          cy.getAttached(".Select").within(() => {
            cy.getAttached(".Select-control").click();
            cy.getAttached(".Select-menu-outer").within(() => {
              cy.findByText(/apples/i).should("exist");
            });
            cy.findByRole("button", {
              name: /manage automations/i,
            }).should("not.exist");
          });
        });
      });
    });
    describe("Query pages", () => {
      beforeEach(() => cy.visit("/queries/manage"));
      it("allows global admin to select teams targets for query", () => {
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.getAttached(".fleet-checkbox__input").check({ force: true });
            });
          cy.findAllByText(/detect presence/i).click();
        });

        cy.getAttached(".query-form__button-wrap").within(() => {
          cy.findByRole("button", { name: /run/i }).click();
        });
        cy.contains("h3", /teams/i).should("exist");
        cy.contains(".selector-name", /apples/i).should("exist");
      });
    });
    describe("Manage schedules page", () => {
      beforeEach(() => cy.visit("/schedule/manage"));
      it("shows inherited queries", () => {
        cy.getAttached(".no-schedule__schedule-button").click();
        // TODO: Unable to add tests because "Schedule a query" button detattaches even when using `getAttached`
      });
    });

    describe("Manage policies page", () => {
      beforeEach(() => cy.visit("/policies/manage"));
      it("allows global admin to click 'Manage automations' button", () => {
        cy.getAttached(".button-wrap")
          .findByRole("button", { name: /manage automations/i })
          .click();
        cy.findByRole("button", { name: /cancel/i }).click();
      });
      it("allows global admin to add a new policy", () => {
        cy.getAttached(".button-wrap")
          .findByRole("button", { name: /add a polic/i })
          .click();
        // Add a default policy
        cy.findByText(/gatekeeper enabled/i).click();
        cy.getAttached(".policy-form__button-wrap--new-policy").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save policy/i }).click();
        });
        cy.findByRole("button", { name: /^Save$/ }).click();
        cy.findByText(/policy created/i).should("exist");
      });
      it("allows global admin to delete a team policy", () => {
        cy.visit("/policies/manage");
        cy.getAttached(".Select-control").within(() => {
          cy.findByText(/all teams/i).click();
        });
        cy.getAttached(".Select-menu")
          .contains(/apples/i)
          .click();
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.getAttached(".fleet-checkbox__input").check({
                force: true,
              });
            });
        });
        cy.findByRole("button", { name: /delete/i }).click();
        cy.getAttached(".remove-policies-modal").within(() => {
          cy.findByRole("button", { name: /delete/i }).should("exist");
          cy.findByRole("button", { name: /cancel/i }).click();
        });
      });
      it("allows global admin to edit a team policy", () => {
        cy.visit("policies/manage");
        cy.findByText(/all teams/i).click();
        cy.findByText(/apples/i).click();
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.getAttached(".fleet-checkbox__input").check({
                force: true,
              });
            });
        });
        cy.findByText(/filevault enabled/i).click();
        cy.getAttached(".policy-form__button-wrap").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save/i }).should("exist");
        });
      });
    });
    describe("Admin settings page", () => {
      beforeEach(() => cy.visit("/settings/organization"));
      it("allows global admin to access team settings", () => {
        cy.getAttached(".react-tabs").within(() => {
          cy.findByText(/teams/i).click();
        });
        // Access the Settings - Team details page
        cy.getAttached("tbody").within(() => {
          cy.findByText(/apples/i).click();
        });
        cy.findByText(/apples/i).should("exist");
        cy.findByText(/manage users with global access here/i).should("exist");
      });
      it("displays the 'Team' section in the create user modal", () => {
        cy.getAttached(".react-tabs").within(() => {
          cy.findByText(/users/i).click();
        });
        cy.findByRole("button", { name: /create user/i }).click();
        cy.findByText(/assign teams/i).should("exist");
      });
    });
    describe("User profile page", () => {
      it("renders elements according to role-based access controls", () => {
        cy.visit("/profile");
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText(/team/i)
            .next()
            .contains(/global/i);
          cy.findByText("Role").next().contains(/admin/i);
        });
      });
    });
  });
});

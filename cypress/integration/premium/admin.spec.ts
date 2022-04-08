describe("Premium tier - Admin user", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedPremium();
    cy.seedQueries();
    cy.seedPolicies("apples");
    cy.addDockerHost("apples"); // host not transferred
    cy.addDockerHost("oranges"); // host transferred between teams by global admin
  });
  after(() => {
    cy.logout();
    cy.stopDockerHost();
  });

  describe("Global admin", () => {
    beforeEach(() => {
      cy.loginWithCySession("anna@organization.com", "user123#");
    });
    describe("Navigation", () => {
      beforeEach(() => cy.visit("/dashboard"));
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
    // Global Admin dashboard tested in integration/free/admin.spec.ts
    // Team Admin dashboard tested below in integration/premium/admin.spec.ts
    describe("Manage hosts page", () => {
      beforeEach(() => cy.visit("/hosts/manage"));
      it("displays team column in hosts table", () => {
        cy.getAttached(".data-table__table th")
          .contains("Team")
          .should("be.visible");
      });
      it("allows global admin to see and click 'Add hosts'", () => {
        cy.getAttached(".button-wrap")
          .contains("button", /add hosts/i)
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
      beforeEach(() => cy.visit("hosts/2"));
      it("allows global admin to transfer host to an existing team", () => {
        cy.getAttached(".host-details__transfer-button").click();
        cy.findByText(/create a team/i).should("exist");
        cy.getAttached(".Select-control").click();
        cy.getAttached(".Select-menu").within(() => {
          cy.findByText(/no team/i).should("exist");
          cy.findByText(/oranges/i).should("exist");
          cy.findByText(/apples/i).click();
        });
        cy.getAttached(".transfer-host-modal__button-wrap")
          .contains("button", /transfer/i)
          .click();
        cy.findByText(/transferred to apples/i).should("exist");
        cy.findByText(/team/i).next().contains("Apples");
      });
      it("allows global admin to create an operating system policy", () => {
        cy.getAttached(".info-flex").within(() => {
          cy.findByText(/ubuntu/i).should("exist");
          cy.getAttached(".host-summary__os-policy-button").click();
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
        cy.getAttached(".delete-host-modal__modal").within(() => {
          cy.findByText(/delete host/i).should("exist");
          cy.contains("button", /delete/i).should("exist");
          cy.getAttached(".modal__ex").click();
        });
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => cy.visit("/software/manage"));
      it("allows global admin to update software vulnerability automation", () => {
        cy.getAttached(".manage-software-page__header-wrap").within(() => {
          cy.getAttached(".Select").within(() => {
            cy.findByText(/all teams/i).should("exist");
          });
          cy.findByRole("button", { name: /manage automations/i }).click();
        });
        cy.getAttached(".manage-automations-modal").within(() => {
          cy.getAttached(".fleet-slider").click();
        });
        cy.getAttached("#webhook-url").click().type("www.foo.com/bar");
        cy.findByRole("button", { name: /^Save$/ }).click();
        // Confirm manage automations webhook was added successfully
        cy.findByText(/updated vulnerability automations/i).should("exist");
        cy.getAttached(".button-wrap").within(() => {
          cy.findByRole("button", {
            name: /manage automations/i,
          }).click();
        });
        cy.getAttached(".manage-automations-modal").within(() => {
          cy.getAttached(".fleet-slider--active").should("exist");
        });
      });
      it("hides manage automations button since all teams not selected", () => {
        cy.getAttached(".manage-software-page__header-wrap").within(() => {
          cy.getAttached(".Select").within(() => {
            cy.findByText(/all teams/i).click();
            cy.findByText(/apples/i).click();
          });
          cy.findByText(/manage automations/i).should("not.exist");
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
    // Global Admin schedule tested in integration/free/admin.spec.ts
    // Team Admin team schedule tested below in integration/premium/admin.spec.ts
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
          .findByRole("button", { name: /add a policy/i })
          .click();
        // Add a default policy
        cy.findByText(/gatekeeper enabled/i).click();
        cy.getAttached(".policy-form__button-wrap").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save policy/i }).click();
        });
        cy.findByRole("button", { name: /^Save$/ }).click();
        cy.findByText(/policy created/i).should("exist");
        cy.findByText(/gatekeeper enabled/i).should("exist");
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
      it("allows global admin to edit existing user password", () => {
        cy.visit("/settings/users");
        cy.getAttached("tbody").within(() => {
          cy.findByText(/oliver@organization.com/i)
            .parent()
            .next()
            .within(() => cy.getAttached(".Select-placeholder").click());
        });
        cy.getAttached(".Select-menu").within(() => {
          cy.findByText(/edit/i).click();
        });
        cy.getAttached(".create-user-form").within(() => {
          cy.findByLabelText(/email/i).should("exist");
          cy.findByLabelText(/password/i).should("exist");
        });
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

  describe("Team admin", () => {
    beforeEach(() => {
      cy.loginWithCySession("anita@organization.com", "user123#");
    });
    describe("Navigation", () => {
      beforeEach(() => cy.visit("/dashboard"));
      it("displays intended team admin top navigation", () => {
        cy.getAttached(".site-nav-container").within(() => {
          cy.findByText(/hosts/i).should("exist");
          cy.findByText(/software/i).should("exist");
          cy.findByText(/queries/i).should("exist");
          cy.findByText(/schedule/i).should("exist");
          cy.findByText(/policies/i).should("exist");
          cy.getAttached(".user-menu").click();
          cy.findByText(/manage users/i).should("not.exist");
          cy.findByText(/settings/i).click();
        });
        cy.getAttached(".react-tabs__tab--selected").within(() => {
          cy.findByText(/members/i).should("exist");
        });
        cy.getAttached(".react-tabs__tab-list").within(() => {
          cy.findByText(/agent options/i).should("exist");
        });
      });
    });
    describe("Dashboard", () => {
      beforeEach(() => cy.visit("/dashboard"));
      it("displays cards for all platforms", () => {
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/apples/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".hosts-status").should("exist");
          cy.getAttached(".home-software").should("exist");
          cy.get(".activity-feed").should("not.exist");
        });
      });
      it("displays cards for windows only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/windows/i).click();
        });
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/apples/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".hosts-status").should("exist");
          // "get" because we expect it not to exist
          cy.get(".home-software").should("not.exist");
          cy.get(".activity-feed").should("not.exist");
        });
      });
      it("displays cards for linux only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/linux/i).click();
        });
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/apples/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".hosts-status").should("exist");
          // "get" because we expect it not to exist
          cy.get(".home-software").should("not.exist");
          cy.get(".activity-feed").should("not.exist");
        });
      });
      it("displays cards for macOS only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/macos/i).click();
        });
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/apples/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".hosts-status").should("exist");
          cy.getAttached(".home-munki").should("exist");
          cy.getAttached(".home-mdm").should("exist");
          // "get" because we expect it not to exist
          cy.get(".home-software").should("not.exist");
          cy.get(".activity-feed").should("not.exist");
        });
      });
      it("views all hosts for all platforms", () => {
        cy.findByText(/view all hosts/i).click();
        cy.get(".manage-hosts__label-block").should("not.exist");
      });
      it("views all hosts for windows only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/windows/i).click();
        });
        cy.findByText(/view all hosts/i).click();
        cy.getAttached(".manage-hosts__label-block").within(() => {
          cy.getAttached(".title").within(() => {
            cy.findByText(/windows/i).should("exist");
          });
        });
      });
      it("views all hosts for linux only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/linux/i).click();
        });
        cy.findByText(/view all hosts/i).click();
        cy.getAttached(".manage-hosts__label-block").within(() => {
          cy.getAttached(".title").within(() => {
            cy.findByText(/linux/i).should("exist");
          });
        });
      });
      it("views all hosts for macOS only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/macos/i).click();
        });
        cy.findByText(/view all hosts/i).click();
        cy.getAttached(".manage-hosts__label-block").within(() => {
          cy.getAttached(".title").within(() => {
            cy.findByText(/macos/i).should("exist");
          });
        });
      });
    });
    describe("Manage hosts page", () => {
      beforeEach(() => {
        cy.visit("/hosts/manage");
      });
      it("displays team column in hosts table", () => {
        cy.getAttached(".data-table__table th")
          .contains("Team")
          .should("be.visible");
      });
      it("allows team admin to see and click 'Add hosts'", () => {
        cy.getAttached(".button-wrap")
          .contains("button", /add hosts/i)
          .click();
        cy.getAttached(".modal__content").contains("button", /done/i).click();
      });
      it("allows team admin to add new enroll secret", () => {
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
      it("allows team admin to create an operating system policy", () => {
        cy.getAttached(".info-flex").within(() => {
          cy.findByText(/ubuntu/i).should("exist");
          cy.getAttached(".host-summary__os-policy-button").click();
        });
        cy.getAttached(".modal__content")
          .findByRole("button", { name: /create new policy/i })
          .should("exist");
      });
      it("allows team admin to query host but not transfer host", () => {
        cy.getAttached(".host-details__query-button").should("exist");
        cy.findByText(/transfer/i).should("not.exist");
      });
      it("allows team admin to delete a host", () => {
        cy.getAttached(".host-details__action-button-container")
          .contains("button", /delete/i)
          .click();
        cy.getAttached(".delete-host-modal__modal").within(() => {
          cy.findByText(/delete host/i).should("exist");
          cy.contains("button", /delete/i).should("exist");
          cy.getAttached(".modal__ex").click();
        });
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => cy.visit("/software/manage"));
      it("hides manage automations button since all teams not selected", () => {
        cy.getAttached(".manage-software-page__header-wrap").within(() => {
          cy.findByText(/apples/i).should("exist");
        });
        cy.findByText(/manage automations/i).should("not.exist");
      });
    });
    describe("Query pages", () => {
      beforeEach(() => cy.visit("/queries/manage"));
      it("allows team admin to select teams targets for query", () => {
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
      it("disables team admin from deleting or editing a query not authored by them", () => {
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.getAttached(".fleet-checkbox__input").should("be.disabled");
            });
          cy.findAllByText(/detect presence/i).click();
        });
        cy.getAttached(".query-form__save").should("be.disabled");
      });
    });
    describe("Manage schedules page", () => {
      beforeEach(() => {
        cy.visit("/schedule/manage");
      });
      it("hides advanced button when team admin", () => {
        cy.getAttached(".manage-schedule-page__header-wrap").within(() => {
          cy.findByText(/apples/i).should("exist");
        });
        cy.findByText(/advanced/i).should("not.exist");
      });
      it("creates a new team scheduled query", () => {
        cy.getAttached(".no-schedule__schedule-button").click();
        cy.getAttached(".schedule-editor-modal__form").within(() => {
          cy.findByText(/select query/i).click();
          cy.findByText(/detect presence/i).click();
          cy.getAttached(".schedule-editor-modal__btn-wrap").within(() => {
            cy.findByRole("button", { name: /schedule/i }).click();
          });
        });
        cy.findByText(/successfully added/i).should("be.visible");
      });
      it("edit a team's scheduled query successfully", () => {
        cy.getAttached("tbody>tr")
          .should("have.length", 1)
          .within(() => {
            cy.findByText(/action/i).click();
            cy.findByText(/edit/i).click();
          });
        cy.getAttached(".schedule-editor-modal__form").within(() => {
          cy.findByText(/every day/i).click();
          cy.findByText(/every 6 hours/i).click();

          cy.getAttached(".schedule-editor-modal__btn-wrap").within(() => {
            cy.findByRole("button", { name: /schedule/i }).click();
          });
        });
        cy.findByText(/successfully updated/i).should("be.visible");
      });
      it("remove a team's scheduled query successfully", () => {
        cy.getAttached("tbody>tr")
          .should("have.length", 1)
          .within(() => {
            cy.findByText(/6 hours/i).should("exist");
            cy.findByText(/action/i).click();
            cy.findByText(/remove/i).click();
          });
        cy.getAttached(".remove-scheduled-query-modal__btn-wrap").within(() => {
          cy.findByRole("button", { name: /remove/i }).click();
        });
        cy.findByText(/successfully removed/i).should("be.visible");
      });
    });
    describe("Manage policies page", () => {
      beforeEach(() => cy.visit("/policies/manage"));
      it("hides manage automations button when all teams not selected", () => {
        cy.getAttached(".manage-policies-page__header-wrap").within(() => {
          cy.findByText(/apples/i).should("exist");
        });
        cy.findByText(/manage automations/i).should("not.exist");
      });
      it("allows team admin to add a new policy", () => {
        cy.getAttached(".button-wrap")
          .findByRole("button", { name: /add a policy/i })
          .click();
        // Add a default policy
        cy.findByText(/gatekeeper enabled/i).click();
        cy.getAttached(".policy-form__button-wrap").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save policy/i }).click();
        });
        cy.findByRole("button", { name: /^Save$/ }).click();
        cy.findByText(/policy created/i).should("exist");
      });
      it("allows team admin to edit a team policy", () => {
        cy.visit("policies/manage");
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
      it("allows team admin to delete a team policy", () => {
        cy.visit("/policies/manage");
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
    });
    describe("Team admin settings page", () => {
      beforeEach(() => cy.visit("/settings/teams/1/members"));
      it("allows team admin to access team settings", () => {
        // Access the Settings - Team details page
        cy.findByText(/apples/i).should("exist");
      });
      it("displays the team admin controls", () => {
        cy.findByRole("button", { name: /add member/i }).click();
        cy.findByRole("button", { name: /cancel/i }).click();
        cy.findByRole("button", { name: /add hosts/i }).click();
        cy.findByRole("button", { name: /done/i }).click();
        cy.findByRole("button", { name: /manage enroll secrets/i }).click();
        cy.findByRole("button", { name: /done/i }).click();
      });
      it("allows team admin to edit a team member", () => {
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .eq(1)
            .within(() => {
              cy.findByText(/action/i).click();
              cy.findByText(/edit/i).click();
            });
        });
        cy.getAttached(".select-role-form__role-dropdown").within(() => {
          cy.findByText(/observer/i).click();
          cy.findByText(/maintainer/i).click();
        });
        cy.findByRole("button", { name: /save/i }).click();
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .eq(1)
            .within(() => {
              cy.findByText(/maintainer/i).should("exist");
            });
        });
      });
      it("does not allow team admin to edit existing user password", () => {
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .eq(1)
            .within(() => {
              cy.findByText(/action/i).click();
              cy.findByText(/edit/i).click();
            });
        });
        cy.getAttached(".create-user-form").within(() => {
          cy.findByLabelText(/email/i).should("exist");
          cy.findByLabelText(/password/i).should("not.exist");
        });
      });
      it("allows team admin to edit team name", () => {
        cy.findByRole("button", { name: /edit team/i }).click();
        cy.findByLabelText(/team name/i)
          .clear()
          .type("Mystic");
        cy.findByRole("button", { name: /save/i }).click();
        cy.findByText(/updated team name/i).should("exist");
      });
    });
    describe("User profile page", () => {
      it("should render elements according to role-based access controls", () => {
        cy.visit("/profile");
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText(/team/i)
            .next()
            .contains(/mystic/i); // Updated team name
          cy.findByText("Role").next().contains(/admin/i);
        });
      });
    });
  });
});

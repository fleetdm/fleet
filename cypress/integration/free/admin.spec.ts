describe(
  "Free tier - Admin user",
  {
    defaultCommandTimeout: 20000,
  },
  () => {
    before(() => {
      Cypress.session.clearAllSavedSessions();
      cy.setup();
      cy.loginWithCySession();
      cy.setupSMTP();
      cy.seedFree();
      cy.seedQueries();
      cy.seedPolicies();
      cy.addDockerHost();
    });
    after(() => {
      cy.logout();
      cy.stopDockerHost();
    });

    describe("Navigation", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/dashboard");
      });
      it("displays intended admin top navigation", () => {
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
    describe("Dashboard", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/dashboard");
      });
      it("displays cards for all platforms", () => {
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/fleet test/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".hosts-status").should("exist");
          cy.getAttached(".home-software").should("exist");
          cy.getAttached(".activity-feed").should("exist");
        });
      });
      it("displays cards for windows only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/windows/i).click();
        });
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/fleet test/i).should("exist");
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
          cy.findByText(/fleet test/i).should("exist");
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
          cy.findByText(/fleet test/i).should("exist");
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
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/hosts/manage");
      });
      it("verifies teams is disabled on Manage Host page", () => {
        cy.contains(/team/i).should("not.exist");
      });
      it("allows admin to see and click the 'Generate installer' button", () => {
        cy.findByRole("button", { name: /generate installer/i }).click();
        cy.contains(/team/i).should("not.exist");
        cy.contains("button", /done/i).click();
      });
      it("allows admin to manage and add enroll secret", () => {
        cy.contains("button", /manage enroll secret/i).click();
        cy.contains("button", /add secret/i).click();
        cy.contains("button", /save/i).click();
        cy.contains("button", /done/i).click();
      });
      it("allows admin to open the 'Add label' form", () => {
        cy.findByRole("button", { name: /add label/i }).click();
        cy.findByRole("button", { name: /cancel/i }).click();
      });
    });
    describe("Host details tests", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/hosts/1");
      });
      it("verifies teams is disabled on Host Details page", () => {
        cy.findByText(/team/i).should("not.exist");
        cy.contains("button", /transfer/i).should("not.exist");
      });
      it("allows admin to delete a query", () => {
        cy.findByRole("button", { name: /delete/i }).click();
        cy.findByText(/delete host/i).should("exist");
        cy.findByRole("button", { name: /cancel/i }).click();
      });
      it("allows admin to create a new query", () => {
        cy.findByRole("button", { name: /query/i }).click();
        cy.findByRole("button", { name: /create custom query/i }).should(
          "exist"
        );
        cy.getAttached(".modal__ex").within(() => {
          cy.findByRole("button").click();
        });
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/software/manage");
      });
      it("allows global admin to click 'Manage automations' button", () => {
        cy.getAttached(".manage-software-page__header-wrap").within(() => {
          cy.findByRole("button", { name: /manage automations/i }).click();
        });
        cy.getAttached(".manage-automations-modal__button-wrap").within(() => {
          cy.findByRole("button", {
            name: /cancel/i,
          }).click();
        });
      });
    });
    describe("Query pages", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/queries/manage");
      });
      it("displays the 'Observer can run' column on the queries table", () => {
        cy.contains(/observer can run/i);
      });
      it("allows admin add a new query", () => {
        cy.findByRole("button", { name: /new query/i }).click();
        cy.getAttached(".ace_text-input")
          .click({ force: true })
          .clear({ force: true })
          .type("SELECT * FROM cypress;", {
            force: true,
          });
        cy.findByRole("button", { name: /save/i }).click();
        cy.getAttached(".modal__background").within(() => {
          cy.getAttached(".modal__modal_container").within(() => {
            cy.getAttached(".modal__content").within(() => {
              cy.getAttached("form").within(() => {
                cy.findByLabelText(/name/i).click().type("Cypress test query");
                cy.findByLabelText(/description/i)
                  .click()
                  .type("Cypress test of create new query flow.");
                cy.findByLabelText(/observers can run/i).click({ force: true });
                cy.findByRole("button", { name: /save query/i }).click();
              });
            });
          });
        });
        cy.findByText(/query created/i).should("exist");
      });
      it("allows admin to edit a query", () => {
        cy.findByText(/cypress test query/i).click({ force: true });
        cy.getAttached(".ace_text-input")
          .click({ force: true })
          .clear({ force: true })
          .type("SELECT 1 FROM cypress;", {
            force: true,
          });
        cy.findByText("Save").click(); // we have 'save as new' also
        cy.findByText(/query updated/i).should("exist");
      });
      it("allows admin to run a query", () => {
        cy.findByText(/cypress test query/i).click({ force: true });
        cy.findByText(/run query/i).click({ force: true });
        cy.findByText(/select targets/i).should("exist");
        cy.findByText(/all hosts/i).click();
        cy.findByText(/targets selected/i).should("exist"); // target count
        cy.findByText(/run/i).click();
        cy.findByText(/querying selected hosts/i).should("exist"); // target count
      });
    });
    describe("Manage policies page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/policies/manage");
      });
      it("allows admin to click 'Manage automations' button", () => {
        cy.findByRole("button", { name: /manage automations/i }).click();
        cy.findByRole("button", { name: /cancel/i }).click();
      });
      it("allows admin to add a new policy", () => {
        cy.findByRole("button", { name: /add a policy/i }).click();
        cy.getAttached(".modal__ex").within(() => {
          cy.findByRole("button").click();
        });
      });
      it("allows admin to delete a policy", () => {
        // select checkmark on table
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.getAttached(".fleet-checkbox__input").check({ force: true });
            });
        });
        cy.findByRole("button", { name: /delete/i }).click();
        cy.getAttached(".remove-policies-modal").within(() => {
          cy.findByRole("button", { name: /delete/i }).should("exist");
          cy.findByRole("button", { name: /cancel/i }).click();
        });
      });
      it("allows admin to select a policy and see CTAs to run and save", () => {
        cy.getAttached(".data-table__table").within(() => {
          cy.findByRole("button", { name: /filevault enabled/i }).click();
        });
        cy.getAttached(".policy-form__button-wrap--new-policy").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save/i }).should("exist");
        });
      });
    });
    describe("Admin settings page", () => {
      // cypress tends to fail on uncaught exceptions. since we have
      // our own error handling, it's suggested to use this block to
      // suppress so the tests will keep running
      Cypress.on("uncaught:exception", () => {
        return false;
      });
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/settings/users");
      });
      it("hides access team settings", () => {
        cy.findByText(/teams/i).should("not.exist");
      });
      it("allows admin to access other settings", () => {
        cy.getAttached(".react-tabs").within(() => {
          cy.findByText(/organization settings/i).should("exist");
          cy.findByText(/users/i).click();
        });
      });
      it("displays the 'Create user' button", () => {
        cy.findByRole("button", { name: /create user/i }).click();
      });
      it("hides assigning a user to a team", () => {
        cy.findByText(/team/i).should("not.exist");
      });
      it("verifies admin is not authorized to reach the Team Settings page", () => {
        cy.visit("/settings/teams");
        cy.findByText(/you do not have permissions/i).should("exist");
      });
    });
    describe("User profile page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/profile");
      });
      it("verifies teams is disabled for the Profile page", () => {
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText(/teams/i).should("not.exist");
        });
      });
      it("renders elements according to role-based access controls", () => {
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText("Role").next().contains(/admin/i);
        });
      });
    });
  }
);

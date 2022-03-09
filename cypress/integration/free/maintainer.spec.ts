describe(
  "Free tier - Maintainer user",
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
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/dashboard");
      });
      it("displays intended global maintainer top navigation", () => {
        cy.getAttached(".site-nav-container").within(() => {
          cy.findByText(/hosts/i).should("exist");
          cy.findByText(/software/i).should("exist");
          cy.findByText(/queries/i).should("exist");
          cy.findByText(/schedule/i).should("exist");
          cy.findByText(/policies/i).should("exist");
          cy.getAttached(".user-menu").click();
          cy.findByText(/settings/i).should("not.exist");
          cy.findByText(/manage users/i).should("not.exist");
        });
      });
    });
    describe("Dashboard", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
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
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/hosts/manage");
      });
      it("verifies maintainer is on the Manage Hosts page", () => {
        cy.getAttached(".manage-hosts").within(() => {
          cy.findByText(/edit columns/i).should("exist");
        });
      });
      it("verifies teams is disabled", () => {
        cy.contains(/team/i).should("not.exist");
      });
      it("allows maintainer to see and click the 'Add hosts' button", () => {
        cy.findByRole("button", { name: /add hosts/i }).click();
        cy.contains("button", /done/i).click();
      });
      it("allows maintainer to manage the enroll secret", () => {
        cy.contains("button", /manage enroll secret/i).click();
        cy.contains("button", /done/i).click();
      });
      it("allows maintainer to open the 'Add label' form", () => {
        cy.findByRole("button", { name: /add label/i }).click();
        cy.findByRole("button", { name: /cancel/i }).click();
      });
    });
    describe("Host details tests", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/hosts/1");
      });
      it("verifies teams is disabled", () => {
        cy.findByText(/team/i).should("not.exist");
        cy.contains("button", /transfer/i).should("not.exist");
      });
      it("allows maintainer to delete a host", () => {
        cy.findByRole("button", { name: /delete/i }).click();
        cy.findByText(/delete host/i).should("exist");
        cy.findByRole("button", { name: /cancel/i }).click();
      });
      it("allows maintainer to create a new query on a host", () => {
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
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/software/manage");
      });
      it("allows maintainer to click 'Manage automations' button", () => {
        it("manages software automations when all teams selected", () => {
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
    });
    describe("Query pages", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/queries/manage");
      });
      it("displays the 'Observer can run' column on the queries table", () => {
        cy.contains(/observer can run/i);
      });
      it("allows maintainer to add a new query", () => {
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
                cy.findByLabelText(/observers can run/i).click({
                  force: true,
                });
                cy.findByRole("button", { name: /save query/i }).click();
              });
            });
          });
        });
        cy.findByText(/query created/i).should("exist");
      });
      it("allows maintainer to edit a query", () => {
        cy.findByText(/cypress test query/i).click({ force: true });
        cy.getAttached(".ace_text-input")
          .click({ force: true })
          .clear({ force: true })
          .type("SELECT * FROM cypress;", {
            force: true,
          });
        cy.findByText("Save").click(); // we have 'save as new' also
        cy.findByText(/query updated/i).should("exist");
      });
      it("allows maintainer to run a query", () => {
        cy.findByText(/cypress test query/i).click({ force: true });
        cy.findByText(/run query/i).click({ force: true });
        cy.findByText(/select targets/i).should("exist");
        cy.findByText(/all hosts/i).click();
        cy.findByText(/hosts targeted/i).should("exist"); // target count
        cy.findByText(/run/i).click();
        cy.findByText(/querying selected hosts/i).should("exist"); // target count
      });
    });
    describe("Manage policies page", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/policies/manage");
      });
      it("allows maintainer to manage automations", () => {
        cy.findByRole("button", { name: /manage automations/i }).click();
        cy.findByRole("button", { name: /cancel/i }).click();
      });
      it("allows maintainer to add a policy", () => {
        cy.findByRole("button", { name: /add a policy/i }).click();
        cy.getAttached(".modal__ex").within(() => {
          cy.findByRole("button").click();
        });
      });
      it("allows maintainer to delete a policy", () => {
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
      it("allows maintainer to select a policy and see CTAs to run and save", () => {
        cy.getAttached(".data-table__table").within(() => {
          cy.getAttached("tbody").within(() => {
            cy.getAttached("tr")
              .first()
              .within(() => {
                cy.findByRole("button", {
                  name: /filevault enabled/i,
                }).click();
              });
          });
        });
        cy.getAttached(".policy-form__button-wrap--new-policy").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save/i }).should("exist");
        });
      });
    });
    describe("Manage packs page", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/packs/manage");
      });
      it("allows maintainer to create a pack", () => {
        cy.findByRole("button", { name: /create new pack/i }).click();
        cy.findByLabelText(/name/i).click().type("Errors and crashes");
        cy.findByLabelText(/description/i)
          .click()
          .type("See all user errors and window crashes.");
        cy.findByRole("button", { name: /save query pack/i }).click();
      });
      it("allows maintainer to delete a pack", () => {
        // select checkmark on table
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.getAttached(".fleet-checkbox__input").check({ force: true });
            });
        });
        // cy.get(".fleet-checkbox__input").check({ force: true });
        cy.findByRole("button", { name: /delete/i }).click();
        // Can't figure out how attach findByRole onto modal button
        // Can't use findByText because delete button under modal
        cy.get(".remove-pack-modal__btn-wrap > .button--alert")
          .contains("button", /delete/i)
          .click();
        cy.findByText(/successfully deleted/i).should("be.visible");
        cy.findByText(/server errors/i).should("not.exist");
      });
    });
    describe("User profile page", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
        cy.visit("/profile");
      });
      it("verifies teams is disabled for the Profile page", () => {
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText(/teams/i).should("not.exist");
        });
      });
      it("renders elements according to role-based access controls", () => {
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText("Role")
            .next()
            .contains(/maintainer/i);
        });
      });
    });

    // nav restrictions are at the end because we expect to see a
    // 403 error overlay which will hide the nav and make the test fail
    describe("Nav restrictions", () => {
      // cypress tends to fail on uncaught exceptions. since we have
      // our own error handling, it's suggested to use this block to
      // suppress so the tests will keep running
      Cypress.on("uncaught:exception", () => {
        return false;
      });
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", "user123#");
      });
      it("verifies maintainer does not have access to settings", () => {
        cy.findByText(/settings/i).should("not.exist");
        cy.visit("/settings/organization");
        cy.findByText(/you do not have permissions/i).should("exist");
      });
    });
  }
);

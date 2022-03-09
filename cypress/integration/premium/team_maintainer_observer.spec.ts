describe("Premium tier - Team observer/maintainer user", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedPremium();
    cy.seedQueries();
    cy.seedPolicies("apples");
    cy.addDockerHost("apples");
    cy.addDockerHost("oranges");
  });
  after(() => {
    cy.logout();
    cy.stopDockerHost();
  });
  describe("Team maintainer and team observer", () => {
    beforeEach(() => {
      cy.loginWithCySession("marco@organization.com", "user123#");
    });
    describe("Navigation", () => {
      beforeEach(() => cy.visit("/dashboard"));
      it("displays intended team maintainer and team observer top navigation", () => {
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
      beforeEach(() => cy.visit("/dashboard"));
      it("displays cards for all platforms", () => {
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/apples/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".hosts-status").should("exist");
          cy.getAttached(".home-software").should("exist");
          // "get" because we expect it not to exist
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
  });
  describe("Team observer", () => {
    beforeEach(() => {
      cy.loginWithCySession("marco@organization.com", "user123#");
    });
    describe("Manage hosts page", () => {
      it("should render elements according to role-based access controls", () => {
        cy.visit("/hosts/manage");
        // Hosts table includes teams column
        cy.getAttached(".data-table__table th")
          .contains("Team")
          .should("be.visible");
        cy.findByText(/add label/i).should("not.exist");

        // On observing team, not see the "add hosts" and "Manage enroll secret" buttons
        cy.contains(/apples/i).should("exist");
        cy.contains("button", /add hosts/i).should("not.exist");
        cy.contains("button", /manage enroll secret/i).should("not.exist");
      });
    });
    describe("Manage policies page", () => {
      it("should render elements according to role-based access controls", () => {
        cy.visit("/policies/manage");
        // On observing team, not see the "Add a policy" and "Manage automations" button
        cy.findByText(/apples/i).should("exist");
        cy.findByText(/manage automations/i).should("not.exist");
        cy.findByText(/add a policy/i).should("not.exist");
      });
    });
    describe("Policy detail page", () => {
      it("should render elements according to role-based access controls", () => {
        cy.visit("/policies/manage");
        // Navigate to policy detail page for first policy in manage policies table
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.contains(".fleet-checkbox__input").should("not.exist");
              cy.findByText(/filevault enabled/i).click();
            });
        });
        cy.getAttached(".policy-form__wrapper").within(() => {
          cy.findByRole("button", { name: /run/i }).should("not.exist");
          cy.findByRole("button", { name: /save/i }).should("not.exist");
        });
      });
    });
    // nav restrictions are at the end because we expect to see a
    // 403 error overlay which will hide the nav and make the test fail
    describe("Nav restrictions", () => {
      it("should restrict navigation according to role-based access controls", () => {
        cy.visit("/dashboard");
        cy.findByText(/settings/i).should("not.exist");
        cy.findByText(/schedule/i).should("exist");
        cy.visit("/settings/organization");
        cy.findByText(/you do not have permissions/i).should("exist");
        cy.visit("/packs/manage");
        cy.findByText(/you do not have permissions/i).should("exist");
      });
    });
  });

  describe("Team maintainer", () => {
    // cypress tends to fail on uncaught exceptions. since we have
    // our own error handling, it's suggested to use this block to
    // suppress so the tests will keep running
    Cypress.on("uncaught:exception", () => {
      return false;
    });

    beforeEach(() => {
      cy.loginWithCySession("marco@organization.com", "user123#");
      cy.visit("/hosts/manage");
    });
    describe("Manage hosts page", () => {
      it("should render elements according to role-based access controls", () => {
        // Hosts table includes teams column
        cy.getAttached(".data-table__table th")
          .contains("Team")
          .should("be.visible");
        cy.findByText(/add label/i).should("not.exist");

        // On maintaining team, see the "add hosts" and "Manage enroll secret" buttons
        cy.getAttached(".manage-hosts__header").within(() => {
          cy.contains("Apples").click({ force: true });
          cy.contains("Oranges").click({ force: true });
        });
        cy.contains(/oranges/i);
        cy.getAttached(".button-wrap")
          .contains("button", /add hosts/i)
          .click();
        cy.getAttached(".modal__content").contains("button", /done/i).click();

        // On maintaining team, add new enroll secret
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
    describe("Manage software page", () => {
      beforeEach(() => cy.visit("/software/manage"));
      it("hides manage automations button", () => {
        cy.getAttached(".manage-software-page__header-wrap").within(() => {
          cy.findByRole("button", { name: /manage automations/i }).should(
            "not.exist"
          );
        });
      });
    });
    describe("Manage schedule page", () => {
      it("should render elements according to role-based access controls", () => {
        cy.visit("/schedule/manage");
        cy.contains(/oranges/i).should("exist");
        cy.contains(/advanced/i).should("not.exist");
        cy.getAttached(".no-schedule__cta-buttons").within(() => {
          cy.findByRole("button", { name: /schedule a query/i }).click();
        });
        // Schedule a query on maintaining team
        cy.getAttached(".schedule-editor-modal__form").within(() => {
          cy.findByText(/select query/i).click();
          cy.findByText(/detect presence/i).click();
          cy.findByText(/every day/i).click();
          cy.findByText(/every 6 hours/i).click();
          cy.getAttached(".schedule-editor-modal__btn-wrap").within(() => {
            cy.findByRole("button", { name: /schedule/i }).click();
          });
        });
        cy.findByText(/successfully added/i).should("be.visible");
        cy.getAttached("tbody>tr").should("have.length", 1);
      });
    });
    describe("Manage policies page", () => {
      it("should render elements according to role-based access controls", () => {
        cy.visit("/policies/manage");
        // Switch to from team apples to team oranges
        cy.findByText(/apples/i).click();
        cy.findByText(/oranges/i).click();

        // On maintaining team, not see the "Manage automations" button
        cy.findByText(/manage automations/i).should("not.exist");
        // On maintaining team, should see "add a policy" and "save" a policy
        cy.findByText(/add a policy/i).click();

        // Add a default policy
        cy.findByText(/gatekeeper enabled/i).click();
        cy.getAttached(".policy-form__button-wrap--new-policy").within(() => {
          cy.findByRole("button", { name: /save policy/i }).click();
        });
        cy.findByRole("button", { name: /^Save$/ }).click();
        cy.findByText(/policy created/i).should("exist");

        // On maintaining team, should see "save" and "run" for a new policy
        cy.getAttached(".policy-form__button-wrap--new-policy").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save/i }).should("exist");
        });
      });
    });
    describe("User profile page", () => {
      it("should render elements according to role-based access controls", () => {
        cy.visit("/profile");
        // See 2 Teams in the Team section and Various in the Role section
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText("Teams")
            .next()
            .contains(/2 teams/i);
          cy.findByText("Role")
            .next()
            .contains(/various/i);
        });
      });
    });
    // nav restrictions are at the end because we expect to see a
    // 403 error overlay which will hide the nav and make the test fail
    describe("Nav restrictions", () => {
      it("should restrict navigation according to role-based access controls", () => {
        cy.visit("/dashboard");

        cy.contains("h2", "Hosts").should("exist");
        cy.getAttached("nav").within(() => {
          cy.findByText(/hosts/i).should("exist");
          cy.findByText(/queries/i).should("exist");
          cy.findByText(/schedule/i).should("exist");
          cy.findByText(/packs/i).should("not.exist");
          cy.findByText(/settings/i).should("not.exist");
        });
      });
    });
  });
});

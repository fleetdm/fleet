import CONSTANTS from "../../support/constants";
import hostDetailsPage from "../pages/hostDetailsPage";
import manageHostsPage from "../pages/manageHostsPage";

const { GOOD_PASSWORD } = CONSTANTS;

describe("Premium tier - Maintainer user", () => {
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

  describe("Global maintainer", () => {
    beforeEach(() => {
      cy.loginWithCySession("mary@organization.com", GOOD_PASSWORD);
    });
    describe("Navigation", () => {
      beforeEach(() => cy.visit("/dashboard"));
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
      beforeEach(() => cy.visit("/dashboard"));
      it("displays cards for all platforms", () => {
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/all teams/i).should("exist");
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
          cy.findByText(/all teams/i).should("exist");
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
          cy.findByText(/all teams/i).should("exist");
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
          cy.findByText(/all teams/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".hosts-status").should("exist");
          cy.getAttached(".home-mdm").should("exist");
          // "get" because we expect it not to exist
          cy.get(".home-software").should("not.exist");
          cy.get(".activity-feed").should("not.exist");
        });
      });
      it("views all hosts for all platforms", () => {
        cy.findByText(/view all hosts/i).click();
        cy.findByRole("status", { name: /hosts filtered by/i }).should(
          "not.exist"
        );
      });
      it("views all hosts for windows only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/Windows/i).click();
        });
        cy.findByText(/view all hosts/i).click();
        cy.findByRole("status", { name: /hosts filtered by Windows/i }).should(
          "exist"
        );
      });
      it("views all hosts for linux only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/linux/i).click();
        });
        cy.findByText(/view all hosts/i).click();
        cy.findByRole("status", { name: /hosts filtered by Linux/i }).should(
          "exist"
        );
      });
      it("views all hosts for macOS only", () => {
        cy.get(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/macos/i).click();
        });
        cy.findByText(/view all hosts/i).click();
        cy.findByRole("status", { name: /hosts filtered by macOS/i }).should(
          "exist"
        );
      });
    });
    describe("Manage hosts page", () => {
      it("renders elements according to role-based access controls", () => {
        manageHostsPage.visitsManageHostsPage();
        manageHostsPage.includesTeamColumn();
        manageHostsPage.allowsAddHosts();
        manageHostsPage.allowsManageAndAddSecrets();
      });
    });
    describe("Host details page", () => {
      beforeEach(() => {
        hostDetailsPage.visitsHostDetailsPage(1);
      });
      it("allows global maintainer to transfer host to an existing team", () => {
        hostDetailsPage.transfersHost();
      });
      it("allows global maintainer to create an operating system policy", () => {
        hostDetailsPage.createOperatingSystemPolicy();
      });
      it("allows global maintainer to custom query a host", () => {
        hostDetailsPage.queriesHost();
      });
      it("allows global maintainer to delete a host", () => {
        hostDetailsPage.deletesHost();
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => cy.visit("/software/manage"));
      it("hides 'Manage automations' button from global maintainer", () => {
        cy.findByText(/manage automations/i).should("not.exist");
      });
    });
    describe("Query pages", () => {
      beforeEach(() => cy.visit("/queries/manage"));
      it("allows global maintainer to select teams targets for query", () => {
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
    describe("Manage policies page", () => {
      beforeEach(() => cy.visit("/policies/manage"));
      it("hides manage automations button", () => {
        cy.findByText(/manage hosts/i).should("not.exist");
      });
      it("allows global maintainer to add a new policy", () => {
        cy.getAttached(".policies-table__action-button-container")
          .findByRole("button", { name: /add a policy/i })
          .click();
        // Add a default policy
        cy.findByText(/gatekeeper enabled/i).click();
        cy.getAttached(".policy-form__button-wrap").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save/i }).click();
        });
        cy.getAttached(".modal-cta-wrap").within(() => {
          cy.findByRole("button", { name: /save policy/i }).click();
        });
        cy.findByText(/policy created/i).should("exist");
      });
      it("allows global maintainer to delete a team policy", () => {
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
        cy.getAttached(".delete-policy-modal").within(() => {
          cy.findByRole("button", { name: /delete/i }).should("exist");
          cy.findByRole("button", { name: /cancel/i }).click();
        });
      });
      it("allows global maintainer to edit a team policy", () => {
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
    describe("User profile page", () => {
      it("renders elements according to role-based access controls", () => {
        cy.visit("/profile");
        cy.getAttached(".user-side-panel").within(() => {
          cy.findByText(/team/i)
            .next()
            .contains(/global/i);
          cy.findByText("Role")
            .next()
            .contains(/maintainer/i);
        });
      });
    });
  });
});

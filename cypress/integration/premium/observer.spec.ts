import CONSTANTS from "../../support/constants";
import hostDetailsPage from "../pages/hostDetailsPage";
import manageHostsPage from "../pages/manageHostsPage";
import managePacksPage from "../pages/managePacksPage";
import managePoliciesPage from "../pages/managePoliciesPage";
import manageQueriesPage from "../pages/manageQueriesPage";
import manageSchedulePage from "../pages/manageSchedulePage";
import manageSoftwarePage from "../pages/manageSoftwarePage";
import teamsDropdown from "../pages/teamsDropdown";
import userProfilePage from "../pages/userProfilePage";

const { GOOD_PASSWORD } = CONSTANTS;

describe("Premium tier - Observer user", () => {
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

  describe("Global observer", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
    });
    describe("Navigation", () => {
      beforeEach(() => cy.visit("/dashboard"));
      it("displays intended global observer top navigation", () => {
        cy.getAttached(".site-nav-container").within(() => {
          cy.findByText(/hosts/i).should("exist");
          cy.findByText(/software/i).should("exist");
          cy.findByText(/queries/i).should("exist");
          cy.findByText(/schedule/i).should("not.exist");
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
          cy.findByText(/windows/i).click();
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
        cy.getAttached(".homepage__platforms").within(() => {
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
      beforeEach(() => manageHostsPage.visitsManageHostsPage());
      it("renders team elements", () => {
        manageHostsPage.includesTeamDropdown();
        manageHostsPage.includesTeamColumn();
      });
      it("hides 'Add hosts', 'Add label', and 'Manage enroll secrets' buttons", () => {
        manageHostsPage.hidesButton("Add label");
        manageHostsPage.hidesButton("Add hosts");
        manageHostsPage.hidesButton("Manage enroll secret");
      });
    });
    describe("Host details page", () => {
      beforeEach(() => hostDetailsPage.visitsHostDetailsPage(1));
      it("should render elements according to role-based access controls", () => {
        hostDetailsPage.verifiesTeam("Apples");
        hostDetailsPage.hidesButton("Transfer");
        hostDetailsPage.hidesButton("Delete");
        hostDetailsPage.hidesCustomQueryHost();
        hostDetailsPage.hidesCreateOSPolicy();
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => manageSoftwarePage.visitManageSoftwarePage());
      it("hides manage automations button", () => {
        manageSoftwarePage.hidesButton("Manage automations");
      });
    });
    describe("Query pages", () => {
      beforeEach(() => manageQueriesPage.visitManageQueriesPage());
      it("allows global maintainer to select teams targets for query", () => {
        manageQueriesPage.allowsSelectTeamTargets();
      });
    });
    describe("Policies pages", () => {
      beforeEach(() => managePoliciesPage.visitManagePoliciesPage());
      it("should render elements according to role-based access controls", () => {
        // No global policies seeded, placeholder displayed
        cy.findByText(/ask yes or no questions/i).should("exist");
        cy.findByText(/all your hosts/i).should("exist");

        managePoliciesPage.hidesButton("Manage automations");
        managePoliciesPage.hidesButton("Add a policy");

        teamsDropdown.switchTeams("All teams", "Apples");
        managePoliciesPage.hidesButton("Manage automations");
        managePoliciesPage.hidesButton("Add a policy");

        managePoliciesPage.allowsViewPolicyOnly();
      });
    });
  });

  describe("Team observer", () => {
    beforeEach(() => {
      cy.loginWithCySession("toni@organization.com", GOOD_PASSWORD);
    });
    describe("Nav restrictions", () => {
      it("should restrict navigation according to role-based access controls", () => {
        // cypress tends to fail on uncaught exceptions. since we have
        // our own error handling, it's suggested to use this block to
        // suppress so the tests will keep running
        Cypress.on("uncaught:exception", () => {
          return false;
        });
        cy.findByText(/settings/i).should("not.exist");
        cy.findByText(/schedule/i).should("not.exist");
        cy.visit("/settings/organization");
        cy.findByText(/you do not have permissions/i).should("exist");
        managePacksPage.visitsManagePacksPage();
        cy.findByText(/you do not have permissions/i).should("exist");
        manageSchedulePage.visitManageSchedulePage();
        cy.findByText(/you do not have permissions/i).should("exist");
      });
    });
    describe("Manage hosts page", () => {
      it("should render elements according to role-based access controls", () => {
        manageHostsPage.visitsManageHostsPage();
        manageHostsPage.includesTeamColumn();
        manageHostsPage.hidesButton("Add hosts");
        manageHostsPage.hidesButton("Manage enroll secret");
        manageHostsPage.hidesButton("Add label");
      });
    });
    describe("Manage policies page", () => {
      it("hides 'Add a policy'", () => {
        managePoliciesPage.visitManagePoliciesPage();
        managePoliciesPage.hidesButton("Add a policy");
        cy.findByText(/all teams/i).should("not.exist");
      });
    });
    describe("Policy detail page", () => {
      it("allows viewing policies only", () => {
        managePoliciesPage.visitManagePoliciesPage();
        managePoliciesPage.allowsViewPolicyOnly();
      });
    });
    describe("User profile page", () => {
      it("verifies observer role and team", () => {
        userProfilePage.visitUserProfilePage();
        userProfilePage.showRole("Observer", "Apples");
      });
    });
  });
});

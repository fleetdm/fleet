import CONSTANTS from "../../support/constants";
import dashboardPage from "../pages/dashboardPage";
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
      beforeEach(() => dashboardPage.visitsDashboardPage());
      it("displays cards for all platforms and does not filter host platform", () => {
        dashboardPage.displaysCards("All", "premium");
        dashboardPage.verifiesFilteredHostByPlatform("none");
      });
      it("displays cards for windows only and filters hosts by Windows platform", () => {
        dashboardPage.switchesPlatform("Windows");
        dashboardPage.displaysCards("Windows", "premium");
        dashboardPage.verifiesFilteredHostByPlatform("Windows");
      });
      it("displays cards for linux only and filters hosts by Linux platform", () => {
        dashboardPage.switchesPlatform("Linux");
        dashboardPage.displaysCards("Linux", "premium");
        dashboardPage.verifiesFilteredHostByPlatform("Linux");
      });
      it("displays cards for macOS only and filters hosts by macOS platform", () => {
        dashboardPage.switchesPlatform("macOS");
        dashboardPage.displaysCards("macOS", "premium");
        dashboardPage.verifiesFilteredHostByPlatform("macOS");
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
      it("allows global observer to select teams targets for query", () => {
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

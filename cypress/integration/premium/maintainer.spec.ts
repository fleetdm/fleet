import CONSTANTS from "../../support/constants";
import hostDetailsPage from "../pages/hostDetailsPage";
import managePoliciesPage from "../pages/managePoliciesPage";
import manageHostsPage from "../pages/manageHostsPage";
import manageQueriesPage from "../pages/manageQueriesPage";
import manageSoftwarePage from "../pages/manageSoftwarePage";
import teamsDropdown from "../pages/teamsDropdown";
import userProfilePage from "../pages/userProfilePage";
import dashboardPage from "../pages/dashboardPage";

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
      it("renders 'Add hosts', 'Add label', and 'Manage enroll secrets' buttons", () => {
        manageHostsPage.allowsAddLabelForm();
        manageHostsPage.allowsAddHosts();
        manageHostsPage.allowsManageAndAddSecrets();
      });
    });
    describe("Host details page", () => {
      beforeEach(() => {
        hostDetailsPage.visitsHostDetailsPage(1);
      });
      it("allows global maintainer to create an operating system policy", () => {
        hostDetailsPage.allowsCreateOsPolicy();
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => manageSoftwarePage.visitManageSoftwarePage());
      it("hides 'Manage automations' button from global maintainer", () => {
        manageSoftwarePage.hidesButton("Manage automations");
      });
    });
    describe("Query pages", () => {
      beforeEach(() => manageQueriesPage.visitManageQueriesPage());
      it("allows global maintainer to select teams targets for query", () => {
        manageQueriesPage.allowsSelectTeamTargets();
      });
      // TODO: Allowed to delete self-authored query only
    });
    describe("Manage policies page", () => {
      beforeEach(() => managePoliciesPage.visitManagePoliciesPage());
      it("hides manage automations button", () => {
        managePoliciesPage.hidesButton("Manage automations");
      });
      it("allows global maintainer to add a new policy", () => {
        managePoliciesPage.allowsAddDefaultPolicy();
        managePoliciesPage.verifiesAddedDefaultPolicy();
      });
      it("allows global maintainer to delete a team policy", () => {
        teamsDropdown.switchTeams("All teams", "Apples");
        managePoliciesPage.allowsDeletePolicy();
      });
      it("allows global maintainer to edit a team policy", () => {
        teamsDropdown.switchTeams("All teams", "Apples");
        managePoliciesPage.allowsSelectRunSavePolicy("filevault");
      });
    });
    describe("User profile page", () => {
      it("verifies user role and global access", () => {
        userProfilePage.visitUserProfilePage();
        userProfilePage.showRole("Maintainer", "Global");
      });
    });
  });
});

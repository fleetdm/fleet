import CONSTANTS from "../../support/constants";
import dashboardPage from "../pages/dashboardPage";
import hostDetailsPage from "../pages/hostDetailsPage";
import managePoliciesPage from "../pages/managePoliciesPage";
import manageHostsPage from "../pages/manageHostsPage";
import manageQueriesPage from "../pages/manageQueriesPage";
import manageSoftwarePage from "../pages/manageSoftwarePage";
import userProfilePage from "../pages/userProfilePage";

const { GOOD_PASSWORD } = CONSTANTS;

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
        cy.loginWithCySession("anna@organization.com", GOOD_PASSWORD);
        dashboardPage.visitsDashboardPage();
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
        cy.loginWithCySession("anna@organization.com", GOOD_PASSWORD);
        dashboardPage.visitsDashboardPage();
      });
      it("displays cards for all platforms and does not filter host platform", () => {
        dashboardPage.displaysCards("All");
        dashboardPage.verifiesFilteredHostByPlatform("none");
      });
      it("displays cards for windows only and filters hosts by Windows platform", () => {
        dashboardPage.switchesPlatform("Windows");
        dashboardPage.displaysCards("Windows");
        dashboardPage.verifiesFilteredHostByPlatform("Windows");
      });
      it("displays cards for linux only and filters hosts by Linux platform", () => {
        dashboardPage.switchesPlatform("Linux");
        dashboardPage.displaysCards("Linux");
        dashboardPage.verifiesFilteredHostByPlatform("Linux");
      });
      it("displays cards for macOS only and filters hosts by macOS platform", () => {
        dashboardPage.switchesPlatform("macOS");
        dashboardPage.displaysCards("macOS");
        dashboardPage.verifiesFilteredHostByPlatform("macOS");
      });
    });
    describe("Manage hosts page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", GOOD_PASSWORD);
        manageHostsPage.visitsManageHostsPage();
      });
      it("verifies teams is disabled on Manage hosts page", () => {
        manageHostsPage.verifiesTeamsIsDisabled();
      });
      it("allows admin to see and click CTA buttons", () => {
        manageHostsPage.allowsAddLabelForm();
        manageHostsPage.allowsAddHosts();
        manageHostsPage.allowsManageAndAddSecrets();
      });
    });
    describe("Host details page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", GOOD_PASSWORD);
        hostDetailsPage.visitsHostDetailsPage(1);
      });
      it("verifies teams is disabled on Host Details page", () => {
        hostDetailsPage.verifiesTeamsisDisabled();
        hostDetailsPage.hidesButton("Transfer");
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", GOOD_PASSWORD);
        manageSoftwarePage.visitManageSoftwarePage();
      });
      it("allows admin to click 'Manage automations' button", () => {
        manageSoftwarePage.allowsManageAutomations();
      });
    });
    describe("Query pages", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", GOOD_PASSWORD);
        manageQueriesPage.visitManageQueriesPage();
      });
      it("allows admin add a new query", () => {
        manageQueriesPage.allowsCreateNewQuery();
        manageQueriesPage.verifiesCreatedNewQuery();
      });
      it("allows admin to edit a query", () => {
        manageQueriesPage.allowsEditExistingQuery();
        manageQueriesPage.verifiesEditedExistingQuery();
      });
      it("allows admin to run a query", () => {
        manageQueriesPage.allowsRunQuery();
        manageQueriesPage.verifiesRanQuery();
      });
    });
    describe("Manage policies page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", GOOD_PASSWORD);
        managePoliciesPage.visitManagePoliciesPage();
      });
      it("allows admin to click 'Manage automations' button", () => {
        managePoliciesPage.allowsAutomatePolicy();
        managePoliciesPage.verifiesAutomatedPolicy();
      });
      it("allows admin to add a new policy", () => {
        managePoliciesPage.allowsAddDefaultPolicy();
        managePoliciesPage.verifiesAddedDefaultPolicy();
      });
      it("allows admin to delete a policy", () => {
        managePoliciesPage.allowsDeletePolicy();
      });
      it("allows admin to select a policy and see CTAs to run and save", () => {
        managePoliciesPage.allowsSelectRunSavePolicy();
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
        cy.loginWithCySession("anna@organization.com", GOOD_PASSWORD);
        cy.visit("/settings/users");
      });
      it("hides access to Fleet Desktop settings", () => {
        cy.visit("settings/organization");
        cy.findByRole("navigation", { name: "settings" }).within(() => {
          cy.findByText(/organization info/i).should("exist");
          cy.findByText(/fleet desktop/i).should("not.exist");
        });
        cy.visit("settings/organization/fleet-desktop");
        cy.findAllByText(/access denied/i).should("exist");
      });
      it("hides access team settings", () => {
        cy.findByText(/teams/i).should("not.exist");
      });
      it("allows admin to access integrations and users settings", () => {
        cy.getAttached(".react-tabs").within(() => {
          cy.findByText(/organization settings/i).should("exist");
          cy.findByText(/integrations/i).click();
        });
        cy.getAttached(".react-tabs").within(() => {
          cy.findByText(/users/i).click();
        });
      });
      it("displays the 'Create user' button", () => {
        cy.findByRole("button", { name: /create user/i }).click({
          force: true,
        });
      });
      it("hides assigning a user to a team", () => {
        cy.findByText(/team/i).should("not.exist");
      });
      it("allows admin to edit existing user password", () => {
        cy.visit("/settings/users");
        cy.getAttached("tbody").within(() => {
          cy.findByText(/mary@organization.com/i)
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
      it("verifies admin is not authorized to reach the Team Settings page", () => {
        cy.visit("/settings/teams");
        cy.findByText(/you do not have permissions/i).should("exist");
      });
    });
    describe("User profile page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", GOOD_PASSWORD);
        userProfilePage.visitUserProfilePage();
      });
      it("verifies admin role and team", () => {
        userProfilePage.showRole("Admin");
      });
    });
  }
);

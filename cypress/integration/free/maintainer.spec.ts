import CONSTANTS from "../../support/constants";
import dashboardPage from "../pages/dashboardPage";
import hostDetailsPage from "../pages/hostDetailsPage";
import manageHostsPage from "../pages/manageHostsPage";
import manageQueriesPage from "../pages/manageQueriesPage";
import managePacksPage from "../pages/managePacksPage";
import managePoliciesPage from "../pages/managePoliciesPage";
import manageSoftwarePage from "../pages/manageSoftwarePage";
import userProfilePage from "../pages/userProfilePage";

const { GOOD_PASSWORD } = CONSTANTS;

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
        cy.loginWithCySession("mary@organization.com", GOOD_PASSWORD);
        dashboardPage.visitsDashboardPage();
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
        cy.loginWithCySession("mary@organization.com", GOOD_PASSWORD);
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
        cy.loginWithCySession("mary@organization.com", GOOD_PASSWORD);
        manageHostsPage.visitsManageHostsPage();
      });
      it("verifies teams is disabled", () => {
        manageHostsPage.verifiesTeamsIsDisabled();
      });
      it("allows maintainer to see and click 'Add label', 'Add hosts', and 'Manage enroll secrets' buttons", () => {
        manageHostsPage.allowsAddHosts();
        manageHostsPage.allowsManageAndAddSecrets();
      });
    });
    describe("Host details page", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", GOOD_PASSWORD);
        hostDetailsPage.visitsHostDetailsPage(1);
      });
      it("verifies teams is disabled", () => {
        hostDetailsPage.verifiesTeamsisDisabled();
        hostDetailsPage.hidesButton("Transfer");
      });
      it("allows maintainer to create an operating system policy", () => {
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
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", GOOD_PASSWORD);
        manageQueriesPage.visitManageQueriesPage();
      });
      it("allows maintainer to add a new query", () => {
        manageQueriesPage.allowsCreateNewQuery();
        manageQueriesPage.verifiesCreatedNewQuery();
      });
      it("allows maintainer to edit a query", () => {
        manageQueriesPage.allowsEditExistingQuery();
        manageQueriesPage.verifiesEditedExistingQuery();
      });
      it("allows maintainer to run a query", () => {
        manageQueriesPage.allowsRunQuery();
        manageQueriesPage.verifiesRanQuery();
      });
    });
    describe("Manage policies page", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", GOOD_PASSWORD);
        managePoliciesPage.visitManagePoliciesPage();
      });
      it("hides manage automations from maintainer", () => {
        managePoliciesPage.hidesButton("Manage automations");
      });
      it("allows maintainer to add a policy", () => {
        managePoliciesPage.allowsAddDefaultPolicy();
        managePoliciesPage.verifiesAddedDefaultPolicy();
      });
      it("allows maintainer to delete a policy", () => {
        managePoliciesPage.allowsDeletePolicy();
      });
      it("allows maintainer to select a policy and see CTAs to run and save", () => {
        managePoliciesPage.allowsRunSavePolicy();
      });
    });
    /* NOTE: Product decision to remove packs from UI
    describe("Manage packs page", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", GOOD_PASSWORD);
        managePacksPage.visitsManagePacksPage();
      });
      it("allows maintainer to create a pack", () => {
        managePacksPage.allowsCreatePack();
        managePacksPage.verifiesCreatedPack();
      });
      it("allows maintainer to delete a pack", () => {
        managePacksPage.allowsDeletePack();
        managePacksPage.verifiesDeletedPack();
      });
    });
    */
    describe("User profile page", () => {
      beforeEach(() => {
        cy.loginWithCySession("mary@organization.com", GOOD_PASSWORD);
        userProfilePage.visitUserProfilePage();
      });
      it("verifies maintainer role and teams is disabled", () => {
        userProfilePage.showRole("Maintainer");
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
        cy.loginWithCySession("mary@organization.com", GOOD_PASSWORD);
      });
      it("verifies maintainer does not have access to settings", () => {
        cy.findByText(/settings/i).should("not.exist");
        cy.visit("/settings/organization");
        cy.findByText(/you do not have permissions/i).should("exist");
      });
    });
  }
);

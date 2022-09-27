import CONSTANTS from "../../support/constants";
import dashboardPage from "../pages/dashboardPage";
import hostDetailsPage from "../pages/hostDetailsPage";
import manageHostsPage from "../pages/manageHostsPage";
import managePacksPage from "../pages/managePacksPage";
import managePoliciesPage from "../pages/managePoliciesPage";
import manageQueriesPage from "../pages/manageQueriesPage";
import manageSchedulePage from "../pages/manageSchedulePage";
import manageSoftwarePage from "../pages/manageSoftwarePage";
import userProfilePage from "../pages/userProfilePage";

const { GOOD_PASSWORD } = CONSTANTS;

describe("Free tier - Observer user", () => {
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
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      dashboardPage.visitsDashboardPage();
    });
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
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
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
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      manageHostsPage.visitsManageHostsPage();
    });
    it("verifies teams is disabled", () => {
      manageHostsPage.verifiesTeamsIsDisabled();
    });
    it("hides 'Add hosts', 'Add label', and 'Manage enroll secrets' buttons", () => {
      manageHostsPage.hidesButton("Add label");
      manageHostsPage.hidesButton("Add hosts");
      manageHostsPage.hidesButton("Manage enroll secret");
    });
  });
  describe("Host details page", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      hostDetailsPage.visitsHostDetailsPage(1);
    });
    it("verifies teams is disabled on Host Details page", () => {
      hostDetailsPage.verifiesTeamsisDisabled();
    });
    it("hides all cta buttons", () => {
      hostDetailsPage.hidesButton("Transfer");
      hostDetailsPage.hidesButton("Query");
      hostDetailsPage.hidesButton("Delete");
      hostDetailsPage.hidesCreateOSPolicy();
    });
  });
  describe("Manage software page", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      manageSoftwarePage.visitManageSoftwarePage();
    });
    it("hides manage automations button", () => {
      manageSoftwarePage.hidesButton("Manage automations");
    });
  });
  describe("Query page", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      manageQueriesPage.visitManageQueriesPage();
    });
    it("hides 'Create a query' button", () => {
      manageQueriesPage.hidesButton("Create new query");
    });
    it("verifies observer can select a query and only run it", () => {
      cy.getAttached(".data-table__table").within(() => {
        cy.findByRole("button", { name: /detect presence/i }).click();
      });
      cy.findByText(/packs/i).should("not.exist");
      cy.findByLabelText(/query name/i).should("not.exist");
      cy.findByLabelText(/sql/i).should("not.exist");
      cy.findByLabelText(/description/i).should("not.exist");
      cy.findByLabelText(/observer can run/i).should("not.exist");
      cy.findByText(/show sql/i).click();
      cy.findByRole("button", { name: /run query/i }).should("exist");
    });
  });
  describe("Manage policies page", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      managePoliciesPage.visitManagePoliciesPage();
    });
    it("hides manage automations button", () => {
      managePoliciesPage.hidesButton("Manage automations");
    });
    it("hides 'Add a policy' button", () => {
      managePoliciesPage.hidesButton("Add a policy");
    });
    it("hides 'Run', 'Edit', and 'Delete' a policy", () => {
      managePoliciesPage.allowsViewPolicyOnly();
    });
  });
  describe("User profile page", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      userProfilePage.visitUserProfilePage();
    });
    it("verifies user role and teams is disabled", () => {
      userProfilePage.showRole("Observer");
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
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
    });
    it("should restrict navigation according to role-based access controls", () => {
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
});

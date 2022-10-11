import CONSTANTS from "../../support/constants";
import dashboardPage from "../pages/dashboardPage";
import manageHostsPage from "../pages/manageHostsPage";
import managePacksPage from "../pages/managePacksPage";
import managePoliciesPage from "../pages/managePoliciesPage";
import manageSchedulePage from "../pages/manageSchedulePage";
import manageSoftwarePage from "../pages/manageSoftwarePage";
import teamsDropdown from "../pages/teamsDropdown";
import userProfilePage from "../pages/userProfilePage";

const { GOOD_PASSWORD } = CONSTANTS;

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
      cy.loginWithCySession("marco@organization.com", GOOD_PASSWORD);
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
      beforeEach(() => dashboardPage.visitsDashboardPage());
      it("displays cards for all platforms", () => {
        dashboardPage.displaysCards("team", "premium");
      });
      it("displays cards for windows only", () => {
        dashboardPage.switchesPlatform("Windows");
        dashboardPage.displaysCards("Windows", "premium");
      });
      it("displays cards for linux only", () => {
        dashboardPage.switchesPlatform("Linux");
        dashboardPage.displaysCards("Linux", "premium");
      });
      it("displays cards for macOS only", () => {
        dashboardPage.switchesPlatform("macOS");
        dashboardPage.displaysCards("macOS", "premium");
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
  });
  describe("Team observer", () => {
    beforeEach(() => {
      cy.loginWithCySession("marco@organization.com", GOOD_PASSWORD);
    });
    describe("Manage hosts page", () => {
      it("should render elements according to role-based access controls", () => {
        manageHostsPage.visitsManageHostsPage();

        cy.contains(/apples/i).should("exist");
        manageHostsPage.includesTeamColumn();

        manageHostsPage.hidesButton("Add label");
        manageHostsPage.hidesButton("Add hosts");
        manageHostsPage.hidesButton("Manage enroll secrets");
      });
    });
    describe("Manage policies page", () => {
      it("hides 'Manage automation' and 'Add a policy' buttons", () => {
        managePoliciesPage.visitManagePoliciesPage();
        cy.contains(/apples/i).should("exist");

        managePoliciesPage.hidesButton("Manage automations");
        managePoliciesPage.hidesButton("Add a policy");
      });
    });
    describe("Policy detail page", () => {
      it("allows view policy only", () => {
        managePoliciesPage.visitManagePoliciesPage();
        managePoliciesPage.allowsViewPolicyOnly();
      });
    });
    // nav restrictions are at the end because we expect to see a
    // 403 error overlay which will hide the nav and make the test fail
    describe("Nav restrictions", () => {
      it("should restrict navigation according to role-based access controls", () => {
        dashboardPage.visitsDashboardPage();
        cy.findByText(/settings/i).should("not.exist");
        cy.findByText(/schedule/i).should("exist");
        cy.visit("/settings/organization");
        cy.findByText(/you do not have permissions/i).should("exist");
        managePacksPage.visitsManagePacksPage();
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
      cy.loginWithCySession("marco@organization.com", GOOD_PASSWORD);
      manageHostsPage.visitsManageHostsPage();
    });
    describe("Manage hosts page", () => {
      it("should render elements according to role-based access controls", () => {
        manageHostsPage.includesTeamColumn();
        manageHostsPage.hidesButton("Add label");

        // On maintaining team, see the "add hosts" and "Manage enroll secret" buttons
        teamsDropdown.switchTeams("Apples", "Oranges");
        manageHostsPage.includesTeamDropdown("Oranges");
        manageHostsPage.allowsAddHosts();
        manageHostsPage.allowsManageAndAddSecrets();
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => manageSoftwarePage.visitManageSoftwarePage());
      it("hides manage automations button", () => {
        manageSoftwarePage.hidesButton("Manage automations");
      });
    });
    describe("Manage schedule page", () => {
      it("should render elements according to role-based access controls", () => {
        manageSchedulePage.visitManageSchedulePage();
        manageSchedulePage.confirmsTeam("Oranges");
        manageSchedulePage.hidesButton("Advanced");
        manageSchedulePage.allowsAddSchedule();
        manageSchedulePage.verifiesAddedSchedule();
      });
    });
    describe("Manage policies page", () => {
      it("allows team maintainer to add, edit a policy, but not manage automation", () => {
        managePoliciesPage.visitManagePoliciesPage();
        teamsDropdown.switchTeams("Apples", "Oranges");

        managePoliciesPage.hidesButton("Manage automations");
        managePoliciesPage.allowsAddDefaultPolicy();
        managePoliciesPage.verifiesAddedDefaultPolicy();
      });
    });
    describe("User profile page", () => {
      it("verifies user role and team", () => {
        userProfilePage.visitUserProfilePage();
        userProfilePage.showRole("Various", "2 teams");
      });
    });
    // nav restrictions are at the end because we expect to see a
    // 403 error overlay which will hide the nav and make the test fail
    describe("Nav restrictions", () => {
      it("should restrict navigation according to role-based access controls", () => {
        dashboardPage.visitsDashboardPage();

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

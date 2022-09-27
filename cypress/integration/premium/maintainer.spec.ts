import CONSTANTS from "../../support/constants";
import hostDetailsPage from "../pages/hostDetailsPage";
import managePoliciesPage from "../pages/managePoliciesPage";
import manageHostsPage from "../pages/manageHostsPage";
import manageQueriesPage from "../pages/manageQueriesPage";
import manageSoftwarePage from "../pages/manageSoftwarePage";
import teamsDropdown from "../pages/teamsDropdown";
import userProfilePage from "../pages/userProfilePage";

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
      it("allows global maintainer to transfer host to an existing team", () => {
        hostDetailsPage.allowsTransferHost();
        hostDetailsPage.verifiesTransferredHost();
      });
      it("allows global maintainer to create an operating system policy", () => {
        hostDetailsPage.allowsCreateOsPolicy();
      });
      it("allows global maintainer to custom query a host", () => {
        hostDetailsPage.allowsCustomQueryHost();
      });
      it("allows global maintainer to delete a host", () => {
        hostDetailsPage.allowsDeleteHost();
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
        managePoliciesPage.allowsSelectRunSavePolicy();
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

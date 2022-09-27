import CONSTANTS from "../../support/constants";
import hostDetailsPage from "../pages/hostDetailsPage";
import manageHostsPage from "../pages/manageHostsPage";
import managePoliciesPage from "../pages/managePoliciesPage";
import manageQueriesPage from "../pages/manageQueriesPage";
import manageSchedulePage from "../pages/manageSchedulePage";
import manageSoftwarePage from "../pages/manageSoftwarePage";
import userProfilePage from "../pages/userProfilePage";

const { GOOD_PASSWORD } = CONSTANTS;

describe("Premium tier - Team Admin user", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedPremium();
    cy.seedQueries();
    cy.seedPolicies("apples");
    cy.addDockerHost("apples"); // host not transferred
    cy.addDockerHost("oranges"); // host transferred between teams by global admin
  });
  after(() => {
    cy.logout();
    cy.stopDockerHost();
  });

  beforeEach(() => {
    cy.loginWithCySession("anita@organization.com", GOOD_PASSWORD);
  });
  describe("Navigation", () => {
    beforeEach(() => cy.visit("/dashboard"));
    it("displays intended team admin top navigation", () => {
      cy.getAttached(".site-nav-container").within(() => {
        cy.findByText(/hosts/i).should("exist");
        cy.findByText(/software/i).should("exist");
        cy.findByText(/queries/i).should("exist");
        cy.findByText(/schedule/i).should("exist");
        cy.findByText(/policies/i).should("exist");
        cy.getAttached(".user-menu").click();
        cy.findByText(/manage users/i).should("not.exist");
        cy.findByText(/settings/i).click();
      });
      cy.getAttached(".react-tabs__tab--selected").within(() => {
        cy.findByText(/members/i).should("exist");
      });
      cy.getAttached(".react-tabs__tab-list").within(() => {
        cy.findByText(/agent options/i).should("exist");
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
    beforeEach(() => {
      manageHostsPage.visitsManageHostsPage();
    });
    it("should render elements according to role-based access controls", () => {
      manageHostsPage.includesTeamColumn();
      manageHostsPage.allowsAddHosts();
      manageHostsPage.allowsManageAndAddSecrets();
    });
  });
  describe("Host details page", () => {
    beforeEach(() => hostDetailsPage.visitsHostDetailsPage(1));
    it("allows team admin to create an operating system policy", () => {
      hostDetailsPage.createOperatingSystemPolicy();
    });
    it("allows team admin to query host, delete host but not transfer host", () => {
      hostDetailsPage.queriesHost();
      hostDetailsPage.deletesHost();
      hostDetailsPage.hidesButton("Transfer");
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
    it("allows team admin to select teams targets for query", () => {
      manageQueriesPage.allowsSelectTeamTargets();
    });
    it("disables team admin from deleting or editing a query not authored by them", () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.getAttached(".fleet-checkbox__input").should("be.disabled");
          });
        cy.findAllByText(/detect presence/i).click();
      });
      cy.findByRole("button", { name: "Save" }).should("be.disabled");
    });
  });
  describe("Manage schedules page", () => {
    beforeEach(() => {
      manageSchedulePage.visitManageSchedulePage();
    });
    it("hides advanced button when team admin", () => {
      manageSchedulePage.confirmsTeam("Apples");
      manageSchedulePage.hidesButton("Advanced");
    });
    it("creates a new team scheduled query", () => {
      manageSchedulePage.allowsAddSchedule();
      manageSchedulePage.verifiesAddedSchedule();
    });
    it("edit a team's scheduled query successfully", () => {
      manageSchedulePage.allowsEditSchedule();
      manageSchedulePage.verifiesEditedSchedule();
    });
    it("remove a team's scheduled query successfully", () => {
      manageSchedulePage.allowsRemoveSchedule();
      manageSchedulePage.verifiesRemovedSchedule();
    });
  });
  describe("Manage policies page", () => {
    beforeEach(() => managePoliciesPage.visitManagePoliciesPage());
    it("allows team admin to add a new policy", () => {
      managePoliciesPage.allowsAddDefaultPolicy();
      managePoliciesPage.verifiesAddedDefaultPolicy();
    });
    it("allows team admin to edit a team policy", () => {
      managePoliciesPage.visitManagePoliciesPage();
      managePoliciesPage.allowsSelectRunSavePolicy();
    });
    it("allows team admin to automate a team policy", () => {
      managePoliciesPage.allowsAutomatePolicy();
      managePoliciesPage.verifiesAutomatedPolicy();
    });
    it("allows team admin to delete a team policy", () => {
      managePoliciesPage.allowsDeletePolicy();
    });
  });
  describe("Team admin settings page", () => {
    beforeEach(() => cy.visit("/settings/teams/1/members"));
    it("allows team admin to access team settings", () => {
      // Access the Settings - Team details page
      cy.findByText(/apples/i).should("exist");
    });
    it("displays the team admin controls", () => {
      cy.findByRole("button", { name: /create user/i }).click();
      cy.findByRole("button", { name: /cancel/i }).click();
      cy.findByRole("button", { name: /add hosts/i }).click();
      cy.findByRole("button", { name: /done/i }).click();
      cy.findByRole("button", { name: /manage enroll secrets/i }).click();
      cy.findByRole("button", { name: /done/i }).click();
    });
    it("allows team admin to edit a team member", () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .eq(1)
          .within(() => {
            cy.findByText(/action/i).click();
            cy.findByText(/edit/i).click();
          });
      });
      cy.getAttached(".select-role-form__role-dropdown").within(() => {
        cy.findByText(/observer/i).click();
        cy.findByText(/maintainer/i).click();
      });
      cy.findByRole("button", { name: /save/i }).click();
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .eq(1)
          .within(() => {
            cy.findByText(/maintainer/i).should("exist");
          });
      });
    });
    it("allows team admin to edit team name", () => {
      cy.findByRole("button", { name: /edit team/i }).click();
      cy.findByLabelText(/team name/i)
        .clear()
        .type("Mystic");
      cy.findByRole("button", { name: /save/i }).click();
      cy.findByText(/updated team name/i).should("exist");
    });
  });
  describe("User profile page", () => {
    it("should render elements according to role-based access controls", () => {
      userProfilePage.visitUserProfilePage();
      userProfilePage.showRole("Admin", "Mystic");
    });
  });
});

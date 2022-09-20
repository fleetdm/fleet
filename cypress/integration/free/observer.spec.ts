import CONSTANTS from "../../support/constants";
import hostDetailsPage from "../pages/hostDetailsPage";
import manageHostsPage from "../pages/manageHostsPage";

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
      cy.visit("/dashboard");
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
      cy.visit("/dashboard");
    });
    it("displays cards for all platforms", () => {
      cy.getAttached(".homepage__wrapper").within(() => {
        cy.findByText(/fleet test/i).should("exist");
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
        cy.findByText(/fleet test/i).should("exist");
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
        cy.findByText(/fleet test/i).should("exist");
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
        cy.findByText(/fleet test/i).should("exist");
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
      cy.findByRole("status", { name: /hosts filtered by linux/i }).should(
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
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      manageHostsPage.visitsManageHostsPage();
    });
    it("verifies teams is disabled on Manage Host page", () => {
      manageHostsPage.verifiesTeamsIsDisabled();
    });
    it("hides 'Add hosts', 'Add label', and 'Manage enroll secrets' button", () => {
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
    });
  });
  describe("Manage software page", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      cy.visit("/software/manage");
    });
    it("hides manage automations button", () => {
      cy.getAttached(".manage-software-page__header-wrap").within(() => {
        cy.findByRole("button", { name: /manage automations/i }).should(
          "not.exist"
        );
      });
    });
  });
  describe("Query page", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      cy.visit("/queries/manage");
    });
    it("hides create a query button", () => {
      cy.findByRole("button", { name: /create new query/i }).should(
        "not.exist"
      );
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
      cy.visit("/policies/manage");
    });
    it("hides manage automations button", () => {
      cy.findByText(/manage automations/i).should("not.exist");
    });
    it("hides add a policy button", () => {
      cy.findByText(/add a policy/).should("not.exist");
    });
    it("hides run, edit, or delete a policy", () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.get(".fleet-checkbox__input").should("not.exist");
          });
      });
      cy.getAttached(".data-table__table").within(() => {
        cy.findByRole("button", {
          name: /filevault enabled/i,
        }).click();
      });
      cy.getAttached(".policy-form__wrapper").within(() => {
        cy.findByRole("button", { name: /run/i }).should("not.exist");
        cy.findByRole("button", { name: /save/i }).should("not.exist");
      });
    });
  });
  describe("User profile page", () => {
    beforeEach(() => {
      cy.loginWithCySession("oliver@organization.com", GOOD_PASSWORD);
      cy.visit("/profile");
    });
    it("verifies teams is disabled for the Profile page", () => {
      cy.getAttached(".user-side-panel").within(() => {
        cy.findByText(/teams/i).should("not.exist");
      });
    });
    it("renders elements according to role-based access controls", () => {
      cy.getAttached(".user-side-panel").within(() => {
        cy.findByText("Role")
          .next()
          .contains(/observer/i);
      });
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
      cy.visit("/packs/manage");
      cy.findByText(/you do not have permissions/i).should("exist");
      cy.visit("/schedule/manage");
      cy.findByText(/you do not have permissions/i).should("exist");
    });
  });
});

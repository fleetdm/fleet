import CONSTANTS from "../../../support/constants";
import manageSoftwarePage from "../../pages/manageSoftwarePage";

const {
  CONFIG_INTEGRATIONS_AUTOMATIONS,
  CONFIG_INTEGRATIONS_AUTOMATIONS_DISABLED,
} = CONSTANTS;

const enableWebhook = {
  ...CONFIG_INTEGRATIONS_AUTOMATIONS,
  integrations: {
    jira: [
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira1@example.com",
        api_token: "jira123",
        project_key: "PROJECT 1",
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira2@example.com",
        api_token: "jira123",
        project_key: "PROJECT 2",
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
    ],
    zendesk: [
      {
        url: "https://fleetdm.zendesk.com",
        email: "zendesk1@example.com",
        api_token: "zendesk123",
        group_id: 12345678,
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.zendesk.com",
        email: "zendesk2@example.com",
        api_token: "zendesk123",
        group_id: 87654321,
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
    ],
  },
  webhook_settings: {
    host_status_webhook: {
      enable_host_status_webhook: false,
      destination_url: "",
      host_percentage: 0,
      days_count: 0,
    },
    failing_policies_webhook: {
      enable_failing_policies_webhook: false,
      destination_url: "https://www.foo.com/bar",
      policy_ids: [5, 10],
      host_batch_size: 0,
    },
    vulnerabilities_webhook: {
      destination_url: "http://www.foo.com/bar",
      enable_vulnerabilities_webhook: true,
    },
  },
};

const enableJiraSoftwareIntegration = {
  ...CONFIG_INTEGRATIONS_AUTOMATIONS,
  integrations: {
    jira: [
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira1@example.com",
        api_token: "jira123",
        project_key: "PROJECT 1",
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira2@example.com",
        api_token: "jira123",
        project_key: "PROJECT 2",
        enable_failing_policies: false,
        enable_software_vulnerabilities: true,
      },
    ],
    zendesk: [
      {
        url: "https://fleetdm.zendesk.com",
        email: "zendesk1@example.com",
        api_token: "zendesk123",
        group_id: 12345678,
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.zendesk.com",
        email: "zendesk2@example.com",
        api_token: "zendesk123",
        group_id: 87654321,
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
    ],
  },
  webhook_settings: {
    host_status_webhook: {
      enable_host_status_webhook: false,
      destination_url: "",
      host_percentage: 0,
      days_count: 0,
    },
    failing_policies_webhook: {
      enable_failing_policies_webhook: false,
      destination_url: "https://www.foo.com/bar",
      policy_ids: [5, 10],
      host_batch_size: 0,
    },
    vulnerabilities_webhook: {
      destination_url: "http://www.foo.com/bar",
      enable_vulnerabilities_webhook: false,
    },
  },
};

const enableZendeskSoftwareIntegration = {
  ...CONFIG_INTEGRATIONS_AUTOMATIONS,
  integrations: {
    jira: [
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira1@example.com",
        api_token: "jira123",
        project_key: "PROJECT 1",
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira2@example.com",
        api_token: "jira123",
        project_key: "PROJECT 2",
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
    ],
    zendesk: [
      {
        url: "https://fleetdm.zendesk.com",
        email: "zendesk1@example.com",
        api_token: "zendesk123",
        group_id: 12345678,
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.zendesk.com",
        email: "zendesk2@example.com",
        api_token: "zendesk123",
        group_id: 87654321,
        enable_failing_policies: false,
        enable_software_vulnerabilities: true,
      },
    ],
  },
  webhook_settings: {
    host_status_webhook: {
      enable_host_status_webhook: false,
      destination_url: "",
      host_percentage: 0,
      days_count: 0,
    },
    failing_policies_webhook: {
      enable_failing_policies_webhook: false,
      destination_url: "https://www.foo.com/bar",
      policy_ids: [5, 10],
      host_batch_size: 0,
    },
    vulnerabilities_webhook: {
      destination_url: "http://www.foo.com/bar",
      enable_vulnerabilities_webhook: false,
    },
  },
};

describe("Software", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setupWithSoftware();
    cy.loginWithCySession();
    cy.viewport(1600, 900);
  });
  after(() => {
    cy.logout();
  });

  // describe("Manage software page", () => {
  //   beforeEach(() => {
  //     cy.loginWithCySession();
  //     cy.viewport(1600, 900);
  //     manageSoftwarePage.visitManageSoftwarePage();
  //   });
  //   it("renders and searches the host's software,  links to filter hosts by software", () => {
  //     // cy.getAttached(".manage-software-page__count").within(() => {
  //     //   cy.findByText(/902 software items/i).should("exist");
  //     // });
  //     cy.findByPlaceholderText(/search software/i).type("lib");
  //     // Ensures search completes
  //     cy.wait(3000); // eslint-disable-line cypress/no-unnecessary-waiting
  //     cy.getAttached(".table-container__results-count")
  //       .invoke("text")
  //       .then((text) => {
  //         const fullText = text;
  //         const pattern = /[0-9]+/g;
  //         const newCount = fullText.match(pattern);
  //         const searchCount = parseInt(newCount[0], 10);
  //         expect(searchCount).to.be.equal(444);
  //       });
  //     cy.getAttached(".software-link").first().click({ force: true });
  //     cy.getAttached(".manage-hosts__software-filter-block").within(() => {
  //       cy.getAttached(".manage-hosts__software-filter-name-card").should(
  //         "exist"
  //       );
  //     });
  //     cy.getAttached(".table-container__results-count")
  //       .invoke("text")
  //       .then((text) => {
  //         const fullText = text;
  //         const pattern = /[0-9]+/g;
  //         const newCount = fullText.match(pattern);
  //         const searchCount = parseInt(newCount[0], 10);
  //         expect(searchCount).to.be.equal(2);
  //       });
  //   });
  // });
  describe("Manage software page (mock integrations)", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.viewport(1600, 900);
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        CONFIG_INTEGRATIONS_AUTOMATIONS
      ).as("getIntegrations");
      manageSoftwarePage.visitManageSoftwarePage();
      cy.wait("@getIntegrations").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
    });
    it("creates webhook software vulnerability automation", () => {
      cy.getAttached(".manage-software-page__header-wrap").within(() => {
        cy.findByRole("button", { name: /manage automations/i }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-slider").click();
        cy.getAttached(".fleet-slider").click();
        cy.getAttached("#webhook-radio-btn").next().click();
      });
      cy.getAttached("#webhook-url").click().type("http://www.foo.com/bar");
      cy.intercept("PATCH", "/api/latest/fleet/config", enableWebhook).as(
        "createWebhook"
      );
      cy.intercept("GET", "/api/latest/fleet/config", enableWebhook).as(
        "createdWebhook"
      );
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.wait("@createWebhook").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      cy.wait("@createdWebhook").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      // Confirm manage automations webhook was added successfully
      cy.findByText(/updated vulnerability automations/i).should("exist");
      cy.getAttached(".button-wrap").within(() => {
        cy.findByRole("button", {
          name: /manage automations/i,
        }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-slider--active").should("exist");
        cy.getAttached("#webhook-url").should("exist");
      });
    });
    it("creates jira integration software vulnerability automation", () => {
      cy.getAttached(".manage-software-page__header-wrap").within(() => {
        cy.findByRole("button", {
          name: /manage automations/i,
        }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-slider").click();
        cy.getAttached(".fleet-slider").click();
        cy.getAttached("#ticket-radio-btn").next().click();
        cy.findByText(/project 1/i).click();
        cy.findByText(/project 2/i).click();
      });
      cy.intercept(
        "PATCH",
        "/api/latest/fleet/config",
        enableJiraSoftwareIntegration
      ).as("enableJiraSoftwareIntegration");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        enableJiraSoftwareIntegration
      ).as("enabledJiraSoftwareIntegration");
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.wait("@enableJiraSoftwareIntegration").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      // Confirm jira integration was added successfully
      cy.findByText(/updated vulnerability automations/i).should("exist");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        enableJiraSoftwareIntegration
      ).as("getIntegrations");
      manageSoftwarePage.visitManageSoftwarePage();
      cy.wait("@getIntegrations").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      cy.getAttached(".button-wrap").within(() => {
        cy.findByRole("button", {
          name: /manage automations/i,
        }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-slider--active").should("exist");
        cy.findByText(/project 2/i).should("exist");
      });
    });
    it("creates zendesk integration software vulnerability automation", () => {
      cy.getAttached(".manage-software-page__header-wrap").within(() => {
        cy.findByRole("button", {
          name: /manage automations/i,
        }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-slider").click();
        cy.getAttached(".fleet-slider").click();
        cy.getAttached("#ticket-radio-btn").next().click();
        cy.findByText(/project 1/i).click();
        cy.findByText(/87654321/i).click();
      });
      cy.intercept(
        "PATCH",
        "/api/latest/fleet/config",
        enableZendeskSoftwareIntegration
      ).as("enableZendeskSoftwareIntegration");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        enableZendeskSoftwareIntegration
      ).as("enabledZendeskIntegration");
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.wait("@enableZendeskSoftwareIntegration").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      // Confirm zendesk integration was added successfully
      cy.findByText(/updated vulnerability automations/i).should("exist");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        enableZendeskSoftwareIntegration
      ).as("getIntegrations");
      manageSoftwarePage.visitManageSoftwarePage();
      cy.wait("@getIntegrations").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      cy.getAttached(".button-wrap").within(() => {
        cy.findByRole("button", {
          name: /manage automations/i,
        }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-slider--active").should("exist");
        cy.findByText(/87654321/i).should("exist");
      });
    });
    it("disables software vulnerability automation", () => {
      cy.getAttached(".manage-software-page__header-wrap").within(() => {
        cy.findByRole("button", {
          name: /manage automations/i,
        }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-slider").click();
      });
      cy.intercept(
        "PATCH",
        "/api/latest/fleet/config",
        CONFIG_INTEGRATIONS_AUTOMATIONS_DISABLED
      ).as("disableSoftwareAutomations");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        CONFIG_INTEGRATIONS_AUTOMATIONS_DISABLED
      ).as("disabledAutomations");
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.wait("@disableSoftwareAutomations").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      cy.wait("@disabledAutomations").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      // Confirm integration was disabled successfully
      cy.findByText(/updated vulnerability automations/i).should("exist");
      cy.getAttached(".button-wrap").within(() => {
        cy.findByRole("button", {
          name: /manage automations/i,
        }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.findByText(/vulnerability automations disabled/i).should("exist");
      });
    });
  });
});

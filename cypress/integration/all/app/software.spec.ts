const getConfig = {
  org_info: {
    org_name: "Fleet Test",
    org_logo_url: "",
  },
  server_settings: {
    server_url: "https://localhost:8642",
    live_query_disabled: false,
    enable_analytics: true,
    deferred_save_host: false,
  },
  smtp_settings: {
    enable_smtp: false,
    configured: false,
    sender_address: "",
    server: "",
    port: 587,
    authentication_type: "authtype_username_password",
    user_name: "",
    password: "",
    enable_ssl_tls: true,
    authentication_method: "authmethod_plain",
    domain: "",
    verify_ssl_certs: true,
    enable_start_tls: true,
  },
  host_expiry_settings: {
    host_expiry_enabled: true,
    host_expiry_window: 9,
  },
  host_settings: {
    enable_host_users: true,
    enable_software_inventory: true,
  },
  agent_options: {
    config: {
      options: {
        logger_plugin: "tls",
        pack_delimiter: "/",
        logger_tls_period: 10,
        distributed_plugin: "tls",
        disable_distributed: false,
        logger_tls_endpoint: "/api/osquery/log",
        distributed_interval: 10,
        distributed_tls_max_attempts: 3,
      },
      decorators: {
        load: [
          "SELECT uuid AS host_uuid FROM system_info;",
          "SELECT hostname AS hostname FROM system_info;",
        ],
      },
    },
    overrides: {},
  },
  sso_settings: {
    entity_id: "",
    issuer_uri: "",
    idp_image_url: "",
    metadata: "",
    metadata_url: "",
    idp_name: "",
    enable_sso: false,
    enable_sso_idp_login: false,
  },
  vulnerability_settings: {
    databases_path: "",
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
      destination_url: "ok.com",
      policy_ids: [5, 10],
      host_batch_size: 0,
    },
    vulnerabilities_webhook: {
      enable_vulnerabilities_webhook: false,
      destination_url: "",
      host_batch_size: 0,
    },
    interval: "24h0m0s",
  },
  integrations: {
    jira: [
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira1@example.com",
        api_token: "jira123",
        project_key: "PROJECT 1",
        enable_failing_policies: false,
        enable_software_vulnerabilities: true,
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
  update_interval: {
    osquery_detail: 3600000000000,
    osquery_policy: 3600000000000,
  },
  vulnerabilities: {
    databases_path: "/tmp/vulndbs",
    periodicity: 3600000000000,
    cpe_database_url: "",
    cve_feed_prefix_url: "",
    current_instance_checks: "auto",
    disable_data_sync: false,
    recent_vulnerability_max_age: 2592000000000000,
  },
  license: {
    tier: "premium",
    organization: "development-only",
    device_count: 100,
    expiration: "2022-06-30T20:00:00-04:00",
    note: "for development only",
  },
  logging: {
    debug: false,
    json: false,
    result: {
      plugin: "filesystem",
      config: {
        status_log_file:
          "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_status",
        result_log_file:
          "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_result",
        enable_log_rotation: false,
        enable_log_compression: false,
      },
    },
    status: {
      plugin: "filesystem",
      config: {
        status_log_file:
          "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_status",
        result_log_file:
          "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_result",
        enable_log_rotation: false,
        enable_log_compression: false,
      },
    },
  },
};

const enableWebhook = {
  ...getConfig,
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
      destination_url: "ok.com",
      policy_ids: [5, 10],
      host_batch_size: 0,
    },
    vulnerabilities_webhook: {
      destination_url: "www.foo.com/bar",
      enable_vulnerabilities_webhook: true,
    },
  },
};

const enableJiraSoftwareIntegration = {
  ...getConfig,
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
      destination_url: "ok.com",
      policy_ids: [5, 10],
      host_batch_size: 0,
    },
    vulnerabilities_webhook: {
      destination_url: "www.foo.com/bar",
      enable_vulnerabilities_webhook: false,
    },
  },
};

const enableZendeskSoftwareIntegration = {
  ...getConfig,
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
      destination_url: "ok.com",
      policy_ids: [5, 10],
      host_batch_size: 0,
    },
    vulnerabilities_webhook: {
      destination_url: "www.foo.com/bar",
      enable_vulnerabilities_webhook: false,
    },
  },
};

const disableAutomations = {
  ...getConfig,
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
      destination_url: "ok.com",
      policy_ids: [5, 10],
      host_batch_size: 0,
    },
    vulnerabilities_webhook: {
      destination_url: "www.foo.com/bar",
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

  describe("Manage software page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.viewport(1600, 900);
      cy.visit("/software/manage");
    });
    it("renders and searches the host's software,  links to filter hosts by software", () => {
      cy.getAttached(".table-container__header-left").within(() => {
        cy.findByText(/902 software items/i).should("exist");
      });
      cy.findByPlaceholderText(/search software/i).type("lib");
      // Ensures search completes
      cy.wait(3000); // eslint-disable-line cypress/no-unnecessary-waiting
      cy.getAttached(".table-container__results-count")
        .invoke("text")
        .then((text) => {
          const fullText = text;
          const pattern = /[0-9]+/g;
          const newCount = fullText.match(pattern);
          const searchCount = parseInt(newCount[0], 10);
          expect(searchCount).to.be.equal(444);
        });
      cy.getAttached(".software-link").first().click({ force: true });
      cy.getAttached(".manage-hosts__software-filter-block").within(() => {
        cy.getAttached(".manage-hosts__software-filter-name-card").should(
          "exist"
        );
      });
      cy.getAttached(".table-container__results-count")
        .invoke("text")
        .then((text) => {
          const fullText = text;
          const pattern = /[0-9]+/g;
          const newCount = fullText.match(pattern);
          const searchCount = parseInt(newCount[0], 10);
          expect(searchCount).to.be.equal(2);
        });
    });
  });
  describe("Manage software page (mock integrations)", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.viewport(1600, 900);
      cy.intercept("GET", "/api/latest/fleet/config", getConfig).as(
        "getIntegrations"
      );
      cy.visit("/software/manage");
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
      cy.getAttached("#webhook-url").click().type("www.foo.com/bar");
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
      cy.visit("/software/manage");
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
      cy.visit("/software/manage");
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
      cy.intercept("PATCH", "/api/latest/fleet/config", disableAutomations).as(
        "disableAutomations"
      );
      cy.intercept("GET", "/api/latest/fleet/config", disableAutomations).as(
        "disabledAutomations"
      );
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.wait("@disableAutomations").then((configStub) => {
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

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
      enable_failing_policies_webhook: true,
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
        enable_software_vulnerabilities: true,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira2@example.com",
        api_token: "jira123",
        project_key: "PROJECT 2",
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira3@example.com",
        api_token: "jira123",
        project_key: "PROJECT 3",
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
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira2@example.com",
        api_token: "jira123",
        project_key: "PROJECT 2",
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira3@example.com",
        api_token: "jira123",
        project_key: "PROJECT 3",
        enable_software_vulnerabilities: false,
      },
    ],
  },
  webhook_settings: {
    vulnerabilities_webhook: {
      destination_url: "www.foo.com/bar",
      enable_vulnerabilities_webhook: true,
    },
  },
};

const enableIntegration = {
  ...getConfig,
  integrations: {
    jira: [
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira1@example.com",
        api_token: "jira123",
        project_key: "PROJECT 1",
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira2@example.com",
        api_token: "jira123",
        project_key: "PROJECT 2",
        enable_software_vulnerabilities: true,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira3@example.com",
        api_token: "jira123",
        project_key: "PROJECT 3",
        enable_software_vulnerabilities: false,
      },
    ],
  },
  webhook_settings: {
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
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira2@example.com",
        api_token: "jira123",
        project_key: "PROJECT 2",
        enable_software_vulnerabilities: false,
      },
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira3@example.com",
        api_token: "jira123",
        project_key: "PROJECT 3",
        enable_software_vulnerabilities: false,
      },
    ],
  },
  webhook_settings: {
    vulnerabilities_webhook: {
      destination_url: "www.foo.com/bar",
      enable_vulnerabilities_webhook: false,
    },
  },
};

describe("Premium tier - Global Admin user", () => {
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
    cy.loginWithCySession("anna@organization.com", "user123#");
  });
  describe("Navigation", () => {
    beforeEach(() => cy.visit("/dashboard"));
    it("displays intended global admin top navigation", () => {
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
  // Global Admin dashboard tested in integration/free/admin.spec.ts
  // Team Admin dashboard tested below in integration/premium/admin.spec.ts
  describe("Manage hosts page", () => {
    beforeEach(() => cy.visit("/hosts/manage"));
    it("displays team column in hosts table", () => {
      cy.getAttached(".data-table__table th")
        .contains("Team")
        .should("be.visible");
    });
    it("allows global admin to see and click 'Add hosts'", () => {
      cy.getAttached(".button-wrap")
        .contains("button", /add hosts/i)
        .click();
      cy.getAttached(".modal__content").contains("button", /done/i).click();
    });
    it("allows global admin to add new enroll secret", () => {
      cy.getAttached(".button-wrap")
        .contains("button", /manage enroll secret/i)
        .click();
      cy.getAttached(".enroll-secret-modal__add-secret")
        .contains("button", /add secret/i)
        .click();
      cy.getAttached(".secret-editor-modal__button-wrap")
        .contains("button", /save/i)
        .click();
      cy.getAttached(".enroll-secret-modal__button-wrap")
        .contains("button", /done/i)
        .click();
    });
  });
  describe("Manage software page", () => {
    beforeEach(() => {
      cy.intercept("GET", "/api/latest/fleet/config", getConfig).as(
        "getIntegrations"
      );
      cy.visit("/software/manage");
      cy.wait("@getIntegrations").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
    });
    it("allows admin to create webhook software vulnerability automation", () => {
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
    it("allows admin to create jira integration software vulnerability automation", () => {
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
      cy.intercept("PATCH", "/api/latest/fleet/config", enableIntegration).as(
        "enableIntegration"
      );
      cy.intercept("GET", "/api/latest/fleet/config", enableIntegration).as(
        "enabledIntegration"
      );
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.wait("@enableIntegration").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      cy.wait("@enabledIntegration").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      // Confirm jira integration was added successfully
      cy.findByText(/updated vulnerability automations/i).should("exist");
      cy.intercept("GET", "/api/latest/fleet/config", enableIntegration).as(
        "getIntegrations"
      );
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
    it("allows admin to disable software vulnerability automation", () => {
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
    it("hides manage automations button since all teams not selected", () => {
      cy.getAttached(".manage-software-page__header-wrap").within(() => {
        cy.getAttached(".Select").within(() => {
          cy.findByText(/all teams/i).click();
          cy.findByText(/apples/i).click();
        });
        cy.findByText(/manage automations/i).should("not.exist");
      });
    });
  });
  describe("Host details page", () => {
    beforeEach(() => cy.visit("hosts/2"));
    it("allows global admin to transfer host to an existing team", () => {
      cy.getAttached(".host-details__transfer-button").click();
      cy.findByText(/create a team/i).should("exist");
      cy.getAttached(".Select-control").click();
      cy.getAttached(".Select-menu").within(() => {
        cy.findByText(/no team/i).should("exist");
        cy.findByText(/oranges/i).should("exist");
        cy.findByText(/apples/i).click();
      });
      cy.getAttached(".transfer-host-modal__button-wrap")
        .contains("button", /transfer/i)
        .click();
      cy.findByText(/transferred to apples/i).should("exist");
      cy.findByText(/team/i).next().contains("Apples");
    });
    it("allows global admin to create an operating system policy", () => {
      cy.getAttached(".info-flex").within(() => {
        cy.findByText(/ubuntu/i).should("exist");
        cy.getAttached(".host-summary__os-policy-button").click();
      });
      cy.getAttached(".modal__content")
        .findByRole("button", { name: /create new policy/i })
        .should("exist");
    });
    it("allows global admin to create a custom query", () => {
      cy.getAttached(".host-details__query-button").click();
      cy.contains("button", /create custom query/i).should("exist");
      cy.getAttached(".modal__ex").click();
    });
    it("allows global admin to delete a host", () => {
      cy.getAttached(".host-details__action-button-container")
        .contains("button", /delete/i)
        .click();
      cy.getAttached(".delete-host-modal__modal").within(() => {
        cy.findByText(/delete host/i).should("exist");
        cy.contains("button", /delete/i).should("exist");
        cy.getAttached(".modal__ex").click();
      });
    });
  });

  describe("Query pages", () => {
    beforeEach(() => cy.visit("/queries/manage"));
    it("allows global admin to select teams targets for query", () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.getAttached(".fleet-checkbox__input").check({ force: true });
          });
        cy.findAllByText(/detect presence/i).click();
      });

      cy.getAttached(".query-form__button-wrap").within(() => {
        cy.findByRole("button", { name: /run/i }).click();
      });
      cy.contains("h3", /teams/i).should("exist");
      cy.contains(".selector-name", /apples/i).should("exist");
    });
  });
  // Global Admin schedule tested in integration/free/admin.spec.ts
  // Team Admin team schedule tested below in integration/premium/admin.spec.ts
  describe("Manage policies page", () => {
    beforeEach(() => cy.visit("/policies/manage"));
    it("allows global admin to add a new policy", () => {
      cy.getAttached(".button-wrap")
        .findByRole("button", { name: /add a policy/i })
        .click();
      // Add a default policy
      cy.findByText(/gatekeeper enabled/i).click();
      cy.getAttached(".policy-form__button-wrap").within(() => {
        cy.findByRole("button", { name: /run/i }).should("exist");
        cy.findByRole("button", { name: /save policy/i }).click();
      });
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.findByText(/policy created/i).should("exist");
      cy.findByText(/gatekeeper enabled/i).should("exist");
    });
    it("allows global admin to automate a global policy", () => {
      cy.getAttached(".button-wrap")
        .findByRole("button", { name: /manage automations/i })
        .click();
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-slider").click();
        cy.getAttached(".fleet-checkbox__input").check({ force: true });
        cy.getAttached("#webhook-url")
          .clear()
          .type("https://example.com/global_admin");
        cy.findByText(/save/i).click();
      });
      cy.findByText(/successfully updated policy automations/i).should("exist");
    });
    it("allows global admin to delete a team policy", () => {
      cy.visit("/policies/manage");
      cy.getAttached(".Select-control").within(() => {
        cy.findByText(/all teams/i).click();
      });
      cy.getAttached(".Select-menu")
        .contains(/apples/i)
        .click();
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.getAttached(".fleet-checkbox__input").check({
              force: true,
            });
          });
      });
      cy.findByRole("button", { name: /delete/i }).click();
      cy.getAttached(".remove-policies-modal").within(() => {
        cy.findByRole("button", { name: /delete/i }).should("exist");
        cy.findByRole("button", { name: /cancel/i }).click();
      });
    });
    it("allows global admin to edit a team policy", () => {
      cy.visit("policies/manage");
      cy.findByText(/all teams/i).click();
      cy.findByText(/apples/i).click();
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.getAttached(".fleet-checkbox__input").check({
              force: true,
            });
          });
      });
      cy.findByText(/filevault enabled/i).click();
      cy.getAttached(".policy-form__button-wrap").within(() => {
        cy.findByRole("button", { name: /run/i }).should("exist");
        cy.findByRole("button", { name: /save/i }).should("exist");
      });
    });
  });
  describe("Admin settings page", () => {
    beforeEach(() => cy.visit("/settings/organization"));
    it("allows global admin to access team settings", () => {
      cy.getAttached(".react-tabs").within(() => {
        cy.findByText(/teams/i).click();
      });
      // Access the Settings - Team details page
      cy.getAttached("tbody").within(() => {
        cy.findByText(/apples/i).click();
      });
      cy.findByText(/apples/i).should("exist");
      cy.findByText(/manage users with global access here/i).should("exist");
    });
    it("displays the 'Team' section in the create user modal", () => {
      cy.getAttached(".react-tabs").within(() => {
        cy.findByText(/users/i).click();
      });
      cy.findByRole("button", { name: /create user/i }).click();
      cy.findByText(/assign teams/i).should("exist");
    });
    it("allows global admin to edit existing user password", () => {
      cy.visit("/settings/users");
      cy.getAttached("tbody").within(() => {
        cy.findByText("Oliver") // case-sensitive
          .parent()
          .next()
          .next()
          .next()
          .next()
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
  });
  describe("User profile page", () => {
    it("renders elements according to role-based access controls", () => {
      cy.visit("/profile");
      cy.getAttached(".user-settings__additional").within(() => {
        cy.findByText(/team/i)
          .next()
          .contains(/global/i);
        cy.findByText("Role").next().contains(/admin/i);
      });
    });
  });
});

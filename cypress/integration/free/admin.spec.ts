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
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/dashboard");
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
        cy.loginWithCySession("anna@organization.com", "user123#");
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
          cy.getAttached(".home-munki").should("exist");
          cy.getAttached(".home-mdm").should("exist");
          // "get" because we expect it not to exist
          cy.get(".home-software").should("not.exist");
          cy.get(".activity-feed").should("not.exist");
        });
      });
      it("views all hosts for all platforms", () => {
        cy.findByText(/view all hosts/i).click();
        cy.get(".manage-hosts__label-block").should("not.exist");
      });
      it("views all hosts for windows only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/windows/i).click();
        });
        cy.findByText(/view all hosts/i).click();
        cy.getAttached(".manage-hosts__label-block").within(() => {
          cy.getAttached(".title").within(() => {
            cy.findByText(/windows/i).should("exist");
          });
        });
      });
      it("views all hosts for linux only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/linux/i).click();
        });
        cy.findByText(/view all hosts/i).click();
        cy.getAttached(".manage-hosts__label-block").within(() => {
          cy.getAttached(".title").within(() => {
            cy.findByText(/linux/i).should("exist");
          });
        });
      });
      it("views all hosts for macOS only", () => {
        cy.getAttached(".homepage__platforms").within(() => {
          cy.getAttached(".Select-control").click();
          cy.findByText(/macos/i).click();
        });
        cy.findByText(/view all hosts/i).click();
        cy.getAttached(".manage-hosts__label-block").within(() => {
          cy.getAttached(".title").within(() => {
            cy.findByText(/macos/i).should("exist");
          });
        });
      });
    });
    describe("Manage hosts page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/hosts/manage");
      });
      it("verifies teams is disabled on Manage Host page", () => {
        cy.contains(/team/i).should("not.exist");
      });
      it("allows admin to see and click the 'Add hosts' button", () => {
        cy.findByRole("button", { name: /add hosts/i }).click();
        cy.contains("button", /done/i).click();
      });
      it("allows admin to manage and add enroll secret", () => {
        cy.contains("button", /manage enroll secret/i).click();
        cy.contains("button", /add secret/i).click();
        cy.contains("button", /save/i).click();
        cy.contains("button", /done/i).click();
      });
      it("allows admin to open the 'Add label' form", () => {
        cy.findByRole("button", { name: /add label/i }).click();
        cy.findByRole("button", { name: /cancel/i }).click();
      });
    });
    describe("Host details tests", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/hosts/1");
      });
      it("verifies teams is disabled on Host Details page", () => {
        cy.findByText(/team/i).should("not.exist");
        cy.contains("button", /transfer/i).should("not.exist");
      });
      it("allows admin to delete a query", () => {
        cy.findByRole("button", { name: /delete/i }).click();
        cy.findByText(/delete host/i).should("exist");
        cy.findByRole("button", { name: /cancel/i }).click();
      });
      it("allows admin to create a new query", () => {
        cy.findByRole("button", { name: /query/i }).click();
        cy.findByRole("button", { name: /create custom query/i }).should(
          "exist"
        );
        cy.getAttached(".modal__ex").within(() => {
          cy.findByRole("button").click();
        });
      });
    });
    describe("Manage software page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
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
        cy.intercept(
          "PATCH",
          "/api/latest/fleet/config",
          disableAutomations
        ).as("disableAutomations");
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
    describe("Query pages", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/queries/manage");
      });
      it("allows admin add a new query", () => {
        cy.findByRole("button", { name: /new query/i }).click();
        cy.getAttached(".ace_text-input")
          .click({ force: true })
          .clear({ force: true })
          .type("SELECT * FROM cypress;", {
            force: true,
          });
        cy.findByRole("button", { name: /save/i }).click();
        cy.getAttached(".modal__background").within(() => {
          cy.getAttached(".modal__modal_container").within(() => {
            cy.getAttached(".modal__content").within(() => {
              cy.getAttached("form").within(() => {
                cy.findByLabelText(/name/i).click().type("Cypress test query");
                cy.findByLabelText(/description/i)
                  .click()
                  .type("Cypress test of create new query flow.");
                cy.findByLabelText(/observers can run/i).click({ force: true });
                cy.findByRole("button", { name: /save query/i }).click();
              });
            });
          });
        });
        cy.findByText(/query created/i).should("exist");
      });
      it("allows admin to edit a query", () => {
        cy.findByText(/cypress test query/i).click({ force: true });
        cy.getAttached(".ace_text-input")
          .click({ force: true })
          .clear({ force: true })
          .type("SELECT 1 FROM cypress;", {
            force: true,
          });
        cy.findByText("Save").click(); // we have 'save as new' also
        cy.findByText(/query updated/i).should("exist");
      });
      it("allows admin to run a query", () => {
        cy.findByText(/cypress test query/i).click({ force: true });
        cy.findByText(/run query/i).click({ force: true });
        cy.findByText(/select targets/i).should("exist");
        cy.findByText(/all hosts/i).click();
        cy.findByText(/hosts targeted/i).should("exist"); // target count
        cy.findByText(/run/i).click();
        cy.findByText(/querying selected hosts/i).should("exist"); // target count
      });
    });
    describe("Manage policies page", () => {
      beforeEach(() => {
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/policies/manage");
      });
      it("allows admin to click 'Manage automations' button", () => {
        cy.findByRole("button", { name: /manage automations/i }).click();
        cy.findByRole("button", { name: /cancel/i }).click();
      });
      it("allows admin to add a new policy", () => {
        cy.findByRole("button", { name: /add a policy/i }).click();
        cy.getAttached(".modal__ex").within(() => {
          cy.findByRole("button").click();
        });
      });
      it("allows admin to delete a policy", () => {
        // select checkmark on table
        cy.getAttached("tbody").within(() => {
          cy.getAttached("tr")
            .first()
            .within(() => {
              cy.getAttached(".fleet-checkbox__input").check({ force: true });
            });
        });
        cy.findByRole("button", { name: /delete/i }).click();
        cy.getAttached(".remove-policies-modal").within(() => {
          cy.findByRole("button", { name: /delete/i }).should("exist");
          cy.findByRole("button", { name: /cancel/i }).click();
        });
      });
      it("allows admin to select a policy and see CTAs to run and save", () => {
        cy.getAttached(".data-table__table").within(() => {
          cy.findByRole("button", { name: /filevault enabled/i }).click();
        });
        cy.getAttached(".policy-form__button-wrap").within(() => {
          cy.findByRole("button", { name: /run/i }).should("exist");
          cy.findByRole("button", { name: /save/i }).should("exist");
        });
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
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/settings/users");
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
        cy.findByRole("button", { name: /create user/i }).click();
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
        cy.loginWithCySession("anna@organization.com", "user123#");
        cy.visit("/profile");
      });
      it("verifies teams is disabled for the Profile page", () => {
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText(/teams/i).should("not.exist");
        });
      });
      it("renders elements according to role-based access controls", () => {
        cy.getAttached(".user-settings__additional").within(() => {
          cy.findByText("Role").next().contains(/admin/i);
        });
      });
    });
  }
);

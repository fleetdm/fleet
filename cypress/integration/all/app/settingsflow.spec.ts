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
      enable_vulnerabilities_webhook: true,
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

const createConfig = {
  ...getConfig,
  integrations: {
    jira: [
      {
        url: "https://fleetdm.atlassian.com",
        username: "jira@example.com",
        api_token: "jira123",
        project_key: "PROJECT",
        enable_software_vulnerabilities: false,
      },
    ],
  },
};

const editConfig = {
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
        username: "jira0@example.com",
        api_token: "jira0123",
        project_key: "PROJECT 0",
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
};

const deleteConfig = {
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
        username: "jira3@example.com",
        api_token: "jira123",
        project_key: "PROJECT 3",
        enable_software_vulnerabilities: false,
      },
    ],
  },
};

describe("App settings flow", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("Organization settings page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/settings/organization");
    });
    // We're using `scrollBehavior: 'center'` as a default
    // because the sticky header blocks the elements.
    it("edits existing app settings", { scrollBehavior: "center" }, () => {
      cy.getAttached(".app-config-form").within(() => {
        cy.findByLabelText(/organization name/i)
          .clear()
          .type("TJ's Run");
      });

      cy.findByLabelText(/organization avatar url/i)
        .click()
        .type("http://tjsrun.com/img/logo.png");

      cy.findByLabelText(/fleet app url/i)
        .clear()
        .type("https://localhost:5000");

      cy.findByLabelText(/enable single sign on/i).check({ force: true });

      cy.findByLabelText(/identity provider name/i)
        .click()
        .type("Rachel");

      cy.findByLabelText(/entity id/i)
        .click()
        .type("my entity id");

      cy.findByLabelText(/issuer uri/i)
        .click()
        .type("my issuer uri");

      cy.findByLabelText(/idp image url/i)
        .click()
        .type("https://http.cat/100");

      // specifically targeting this one to avoid conflict
      // with cypress seeing multiple "metadata url" - one
      // in a tooltip, the other as the actual label
      cy.getAttached("[for='metadataURL']")
        .click()
        .type("http://github.com/fleetdm/fleet");

      cy.findByLabelText(/allow sso login initiated/i).check({ force: true });

      cy.findByLabelText(/enable smtp/i).check({ force: true });

      cy.findByLabelText(/sender address/i)
        .click()
        .type("rachel@example.com");

      // specifically targeting this one to avoid conflict
      // with cypress seeing multiple "metadata" - one
      // in a tooltip, the other as the actual label
      cy.getAttached("[for='smtpServer']").click().type("localhost");

      cy.getAttached("#smtpPort").clear().type("1025");

      cy.findByLabelText(/use ssl\/tls/i).check({ force: true });

      cy.findByLabelText(/smtp username/i)
        .click()
        .type("rachelsusername");

      cy.findByLabelText(/smtp password/i)
        .click()
        .type("rachelspassword");

      cy.findByLabelText(/enable host status webhook/i).check({
        force: true,
      });

      cy.findByLabelText(/destination url/i)
        .click()
        .type("http://server.com/example");

      cy.getAttached(
        ".app-config-form__host-percentage .Select-control"
      ).click();
      cy.getAttached(".Select-menu-outer").contains(/5%/i).click();

      cy.getAttached(".app-config-form__days-count .Select-control").click();
      cy.getAttached(".Select-menu-outer")
        .contains(/7 days/i)
        .click();

      cy.findByLabelText(/domain/i)
        .click()
        .type("http://www.fleetdm.com");

      cy.findByLabelText(/verify ssl certs/i).check({ force: true });
      cy.findByLabelText(/enable starttls/i).check({ force: true });
      cy.getAttached("[for='enableHostExpiry']").within(() => {
        cy.getAttached("[type='checkbox']").check({ force: true });
      });

      // specifically targeting this one to avoid conflict
      // with cypress seeing multiple "host expiry" - one
      // in the checkbox above, the other as this label
      cy.getAttached("[name='hostExpiryWindow']").clear().type("5");

      cy.findByLabelText(/disable live queries/i).check({ force: true });

      cy.findByRole("button", { name: /update settings/i })
        .invoke("attr", "disabled", false)
        .click();

      cy.findByText(/updated settings/i).should("exist");

      // confirm edits
      cy.visit("/settings/organization");

      cy.getAttached(".app-config-form").within(() => {
        cy.findByLabelText(/organization name/i).should(
          "have.value",
          "TJ's Run"
        );
      });

      cy.findByLabelText(/organization avatar url/i).should(
        "have.value",
        "http://tjsrun.com/img/logo.png"
      );

      cy.findByLabelText(/fleet app url/i).should(
        "have.value",
        "https://localhost:5000"
      );

      cy.findByLabelText(/identity provider name/i).should(
        "have.value",
        "Rachel"
      );

      cy.findByLabelText(/entity id/i).should("have.value", "my entity id");

      cy.findByLabelText(/issuer uri/i).should("have.value", "my issuer uri");

      cy.findByLabelText(/idp image url/i).should(
        "have.value",
        "https://http.cat/100"
      );

      cy.getAttached("#metadataURL").should(
        "have.value",
        "http://github.com/fleetdm/fleet"
      );

      cy.findByLabelText(/sender address/i).should(
        "have.value",
        "rachel@example.com"
      );

      cy.getAttached("#smtpServer").should("have.value", "localhost");

      cy.getAttached("#smtpPort").should("have.value", "1025");

      cy.findByLabelText(/smtp username/i).should(
        "have.value",
        "rachelsusername"
      );

      cy.findByLabelText(/destination url/i).should(
        "have.value",
        "http://server.com/example"
      );

      cy.findByText(/5%/i).should("exist");

      cy.findByText(/7 days/i).should("exist");
      cy.findByText(/1 day/i).should("not.exist");
      cy.findByText(/select one/i).should("not.exist");

      cy.findByLabelText(/host expiry window/i).should("have.value", "5");

      cy.getEmails().then((response) => {
        expect(response.body.items[0].To[0]).to.have.property("Domain");
        expect(response.body.items[0].To[0].Mailbox).to.equal("admin");
        expect(response.body.items[0].To[0].Domain).to.equal("example.com");
        expect(response.body.items[0].From.Mailbox).to.equal("rachel");
        expect(response.body.items[0].From.Domain).to.equal("example.com");
        expect(response.body.items[0].Content.Headers.Subject[0]).to.equal(
          "Hello from Fleet"
        );
      });
    });
  });

  describe("Integrations settings page (empty)", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/settings/integrations");
    });
    it("creates a new jira integration", () => {
      cy.getAttached(".no-integrations__create-button").click();
      cy.getAttached("#url").click().type("https://fleetdm.atlassian.com");
      cy.getAttached("#username").click().type("jira@example.com");
      cy.getAttached("#apiToken").click().type("jira123");
      cy.getAttached("#projectKey").click().type("PROJECT");
      cy.intercept("PATCH", "/api/latest/fleet/config", createConfig).as(
        "createIntegration"
      );
      cy.intercept("GET", "/api/latest/fleet/config", createConfig).as(
        "createdIntegration"
      );
      cy.findByRole("button", { name: /save/i }).click();
      cy.wait("@createIntegration").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
      });
      cy.wait("@createdIntegration").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
      });
      cy.findByText(/successfully added/i).should("exist");
      cy.getAttached(".table-container").within(() => {
        cy.findByText(/fleetdm.atlassian.com - PROJECT/i).should("exist");
      });
    });
  });

  describe("Integrations settings page (seeded)", () => {
    beforeEach(() => {
      Cypress.session.clearAllSavedSessions();
      cy.setup();
      cy.loginWithCySession();
      cy.viewport(1200, 660);
      cy.intercept("GET", "/api/latest/fleet/config", getConfig).as(
        "getIntegrations"
      );
      cy.visit("/settings/integrations");
      cy.wait("@getIntegrations").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
      });
    });
    it("edits jira integration", () => {
      cy.getAttached("tbody>tr")
        .should("have.length", 3)
        .eq(1)
        .within(() => {
          cy.findByText(/action/i).click();
          cy.findByText(/edit/i).click();
        });
      cy.findByLabelText(/jira site url/i)
        .clear()
        .type("https://fleetdm.atlassian.com");
      cy.findByLabelText(/jira username/i)
        .clear()
        .type("jira0@example.com");
      cy.findByLabelText(/jira api token/i)
        .clear()
        .type("jira0123");
      cy.findByLabelText(/jira project key/i)
        .clear()
        .type("PROJECT 0");
      cy.intercept("PATCH", "/api/latest/fleet/config", editConfig).as(
        "editIntegration"
      );
      cy.intercept("GET", "/api/latest/fleet/config", editConfig).as(
        "editedIntegration"
      );
      cy.getAttached(".integration-form__btn-wrap")
        .contains("button", /save/i)
        .click();
      cy.wait("@editIntegration").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
      });
      cy.wait("@editedIntegration").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
        cy.findByText(/successfully edited/i).should("exist");
        cy.getAttached("tbody>tr")
          .should("have.length", 3)
          .eq(0)
          .within(() => {
            cy.findByText(/fleetdm.atlassian.com - project 0/i).should("exist");
          });
      });
    });
    it("deletes jira integration", () => {
      cy.getAttached("tbody>tr")
        .eq(1)
        .within(() => {
          cy.findByText(/project 2/i).should("exist");
          cy.findByText(/action/i).click();
          cy.findByText(/delete/i).click();
        });
      cy.intercept("PATCH", "/api/latest/fleet/config", deleteConfig).as(
        "deleteIntegration"
      );
      cy.intercept("GET", "/api/latest/fleet/config", deleteConfig).as(
        "deletedIntegration"
      );
      cy.getAttached(".delete-integration-modal__btn-wrap")
        .contains("button", /delete/i)
        .click();
      cy.wait("@deleteIntegration").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
      });
      cy.wait("@deletedIntegration").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
      });
      cy.findByText(/successfully deleted/i).should("exist");
      cy.getAttached("tbody>tr").should("have.length", 2);
      cy.findByText(/project 2/i).should("not.exist");
    });
  });
});

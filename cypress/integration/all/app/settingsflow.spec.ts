import CONSTANTS from "../../../support/constants";

const { CONFIG_INTEGRATIONS_AUTOMATIONS } = CONSTANTS;

const addJiraIntegration = {
  ...CONFIG_INTEGRATIONS_AUTOMATIONS,
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

const deleteJiraIntegration = {
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
};

const addZendeskIntegration = {
  ...CONFIG_INTEGRATIONS_AUTOMATIONS,
  integrations: {
    zendesk: [
      {
        url: "https://fleetdm.zendesk.com",
        email: "zendesk1@example.com",
        api_token: "zendesk123",
        group_id: 12345678,
        enable_failing_policies: false,
        enable_software_vulnerabilities: false,
      },
    ],
  },
};

const deleteZendeskIntegration = {
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
    it("edits organization info", () => {
      cy.getAttached(".app-config-form").within(() => {
        cy.findByLabelText(/organization name/i)
          .clear()
          .type("TJ's Run");
        cy.findByLabelText(/organization avatar url/i)
          .click()
          .type("http://tjsrun.com/img/logo.png");
      });

      cy.findByRole("button", { name: /save/i })
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
    });

    it("edits fleet web address", () => {
      cy.findByText(/fleet web address/i).click();

      cy.findByLabelText(/fleet app url/i)
        .clear()
        .type("https://localhost:5000");

      cy.findByRole("button", { name: /save/i })
        .invoke("attr", "disabled", false)
        .click();

      cy.findByText(/updated settings/i).should("exist");

      // confirm edits
      cy.visit("/settings/organization");
      cy.findByText(/fleet web address/i).click();
      cy.findByLabelText(/fleet app url/i).should(
        "have.value",
        "https://localhost:5000"
      );
    });

    it("edits single sign-on settings", () => {
      cy.findByText(/single sign-on options/i).click();
      cy.findByLabelText(/enable single sign-on/i).check({ force: true });

      cy.findByLabelText(/identity provider name/i)
        .click({ force: true })
        .type("Rachel");

      cy.findByLabelText(/entity id/i)
        .click({ force: true })
        .type("my entity id");

      cy.findByLabelText(/idp image url/i)
        .click()
        .type("https://http.cat/100");

      // specifically targeting this one to avoid conflict
      // with cypress seeing multiple "metadata url" - one
      // in a tooltip, the other as the actual label
      cy.getAttached("[for='metadataUrl']")
        .click()
        .type("http://github.com/fleetdm/fleet");

      cy.findByLabelText(/allow sso login initiated/i).check({ force: true });

      cy.findByRole("button", { name: /save/i })
        .invoke("attr", "disabled", false)
        .click();

      cy.findByText(/updated settings/i).should("exist");

      // confirm edits
      cy.visit("/settings/organization");
      cy.findByText(/single sign-on options/i).click();
      cy.findByLabelText(/identity provider name/i).should(
        "have.value",
        "Rachel"
      );

      cy.findByLabelText(/entity id/i).should("have.value", "my entity id");

      cy.findByLabelText(/idp image url/i).should(
        "have.value",
        "https://http.cat/100"
      );

      cy.getAttached("#metadataUrl").should(
        "have.value",
        "http://github.com/fleetdm/fleet"
      );
    });

    it("edits smtp settings", () => {
      cy.findByText(/smtp options/i).click();
      cy.findByLabelText(/enable smtp/i).check({ force: true });

      cy.findByLabelText(/sender address/i)
        .click({ force: true })
        .type("rachel@example.com");

      // specifically targeting this one to avoid conflict
      // with cypress seeing multiple "metadata" - one
      // in a tooltip, the other as the actual label
      cy.findByLabelText(/SMTP server/)
        .click({ force: true })
        .type("localhost");

      cy.getAttached("#smtpPort").clear().type("1025");

      cy.findByLabelText(/use ssl\/tls/i).check({ force: true });

      cy.findByLabelText(/smtp username/i)
        .click()
        .type("rachelsusername");

      cy.findByLabelText(/smtp password/i)
        .click()
        .type("rachelspassword");

      cy.findByRole("button", { name: /save/i })
        .invoke("attr", "disabled", false)
        .click();

      cy.findByText(/updated settings/i).should("exist");

      // confirm edits
      cy.visit("/settings/organization");
      cy.findByText(/smtp options/i).click();
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
      cy.findByText(/single sign-on options/i).click();

      cy.getAttached("#metadataUrl").should(
        "have.value",
        "http://github.com/fleetdm/fleet"
      );
    });

    it("edits host status webhook", () => {
      cy.findByText(/host status webhook/i).click();
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

      cy.findByRole("button", { name: /save/i })
        .invoke("attr", "disabled", false)
        .click();

      cy.findByText(/updated settings/i).should("exist");

      // confirm edits
      cy.visit("/settings/organization");
      cy.findByText(/host status webhook/i).click();

      cy.findByLabelText(/destination url/i).should(
        "have.value",
        "http://server.com/example"
      );

      cy.findByText(/5%/i).should("exist");

      cy.findByText(/7 days/i).should("exist");
      cy.findByText(/1 day/i).should("not.exist");
      cy.findByText(/select one/i).should("not.exist");
    });

    it("edits usage statistics", () => {
      cy.findByText(/usage statistics/i).click();
      cy.findByLabelText(/enable usage statistics/i).check({
        force: true,
      });

      cy.findByRole("button", { name: /save/i })
        .invoke("attr", "disabled", false)
        .click();

      cy.findByText(/updated settings/i).should("exist");

      // confirm edits
      cy.visit("/settings/organization");
      cy.findByText(/usage statistics/i).click();
      cy.findByLabelText(/enable usage statistics/i).should("be.checked");
    });

    it("edits advanced options", () => {
      cy.findByText(/advanced options/i).click();

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

      cy.findByRole("button", { name: /save/i })
        .invoke("attr", "disabled", false)
        .click();

      cy.findByText(/updated settings/i).should("exist");

      // confirm edits
      cy.visit("/settings/organization");
      cy.findByText(/advanced options/i).click();

      cy.findByLabelText(/verify ssl certs/i).should("be.checked");
      cy.findByLabelText(/enable starttls/i).should("be.checked");
      cy.findByLabelText(/host expiry window/i).should("have.value", "5");

      // confirm smtp configured
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
    it("adds a new jira integration", () => {
      cy.getAttached(".no-integrations__add-button").click();
      cy.getAttached("#url").click().type("https://fleetdm.atlassian.com");
      cy.getAttached("#username").click().type("jira@example.com");
      cy.getAttached("#apiToken").click().type("jira123");
      cy.getAttached("#projectKey").click().type("PROJECT");
      cy.intercept("PATCH", "/api/latest/fleet/config", addJiraIntegration).as(
        "addIntegration"
      );
      cy.intercept("GET", "/api/latest/fleet/config", addJiraIntegration).as(
        "addedIntegration"
      );
      cy.findByRole("button", { name: /save/i }).click();
      cy.wait("@addIntegration").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
      });
      cy.wait("@addedIntegration").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
      });
      cy.findByText(/successfully added/i).should("exist");
      cy.getAttached(".table-container").within(() => {
        cy.findByText(/fleetdm.atlassian.com - PROJECT/i).should("exist");
      });
    });
    it("adds a new zendesk integration", () => {
      cy.getAttached(".no-integrations__add-button").click();
      cy.getAttached(".add-integration-modal__form-field--platform").within(
        () => {
          cy.findByText(/jira/i).click();
          cy.findByText(/zendesk/i).click();
        }
      );
      cy.getAttached("#url").click().type("https://fleetdm.zendesk.com");
      cy.getAttached("#email").click().type("zendesk1@example.com");
      cy.getAttached("#apiToken").click().type("zendesk123");
      cy.getAttached("#groupId").click().type("12345678");
      cy.intercept(
        "PATCH",
        "/api/latest/fleet/config",
        addZendeskIntegration
      ).as("addIntegration");
      cy.intercept("GET", "/api/latest/fleet/config", addZendeskIntegration).as(
        "addedIntegration"
      );
      cy.findByRole("button", { name: /save/i }).click();
      cy.wait("@addIntegration").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
      });
      cy.wait("@addedIntegration").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
      });
      cy.findByText(/successfully added/i).should("exist");
      cy.getAttached(".table-container").within(() => {
        cy.findByText(/fleetdm.zendesk.com - 12345678/i).should("exist");
      });
    });
  });

  describe("Integrations settings page (seeded)", () => {
    beforeEach(() => {
      Cypress.session.clearAllSavedSessions();
      cy.setup();
      cy.loginWithCySession();
      cy.viewport(1200, 660);
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        CONFIG_INTEGRATIONS_AUTOMATIONS
      ).as("getIntegrations");
      cy.visit("/settings/integrations");
      cy.wait("@getIntegrations").then((configStub) => {
        cy.log(JSON.stringify(configStub));
        console.log(JSON.stringify(configStub));
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
      cy.intercept(
        "PATCH",
        "/api/latest/fleet/config",
        deleteJiraIntegration
      ).as("deleteIntegration");
      cy.intercept("GET", "/api/latest/fleet/config", deleteJiraIntegration).as(
        "deletedIntegration"
      );
      cy.getAttached(".delete-integration-modal .modal-cta-wrap")
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
      cy.getAttached("tbody>tr").should("have.length", 3);
      cy.findByText(/project 2/i).should("not.exist");
    });
    it("deletes zendesk integration", () => {
      cy.getAttached("tbody>tr")
        .eq(3)
        .within(() => {
          cy.findByText(/87654321/i).should("exist");
          cy.findByText(/action/i).click();
          cy.findByText(/delete/i).click();
        });
      cy.intercept(
        "PATCH",
        "/api/latest/fleet/config",
        deleteZendeskIntegration
      ).as("deleteIntegration");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        deleteZendeskIntegration
      ).as("deletedIntegration");
      cy.getAttached(".delete-integration-modal .modal-cta-wrap")
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
      cy.getAttached("tbody>tr").should("have.length", 3);
      cy.findByText(/87654321/i).should("not.exist");
    });
  });
});

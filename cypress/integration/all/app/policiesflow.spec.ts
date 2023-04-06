import CONSTANTS from "../../../support/constants";
import managePoliciesPage from "../../pages/managePoliciesPage";

const {
  CONFIG_INTEGRATIONS_AUTOMATIONS,
  CONFIG_INTEGRATIONS_AUTOMATIONS_DISABLED,
} = CONSTANTS;

const enableJiraPoliciesIntegration = {
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
        enable_failing_policies: true,
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
      destination_url: "https://www.foo.com/bar",
      enable_vulnerabilities_webhook: false,
    },
  },
};

const enableZendeskPoliciesIntegration = {
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
        enable_failing_policies: true,
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
      destination_url: "https://www.foo.com/bar",
      enable_vulnerabilities_webhook: false,
    },
  },
};

describe("Policies flow (empty)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("Manage policies page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      managePoliciesPage.visitManagePoliciesPage();
    });
    it("creates a custom policy", () => {
      cy.getAttached(".empty-table__cta-buttons").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.findByText(/create your own policy/i).click();
      cy.getAttached(".policy-page__form .ace_scroller")
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM users WHERE username = 'backup' LIMIT 1;"
        );
      cy.findByRole("button", { name: /save/i }).click();
      cy.getAttached(".policy-form__policy-save-modal-name")
        .click()
        .type("Does the device have a user named 'backup'?");
      cy.getAttached(".policy-form__policy-save-modal-description")
        .click()
        .type("Returns yes or no for having a user named 'backup'");
      cy.getAttached(".policy-form__policy-save-modal-resolution")
        .click()
        .type("Create a user named 'backup'");
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.findByText(/policy created/i).should("exist");
    });

    it("creates a default policy", () => {
      managePoliciesPage.allowsAddDefaultPolicy();
      managePoliciesPage.verifiesAddedDefaultPolicy();
    });
  });

  describe("Platform compatibility", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      managePoliciesPage.visitManagePoliciesPage();
    });
    const platforms = ["macOS", "Windows", "Linux"];

    const testCompatibility = (
      el: JQuery<HTMLElement>,
      i: number,
      expected: boolean[]
    ) => {
      const check = expected[i]
        ? "compatible-platform"
        : "incompatible-platform";
      const compatibility = expected[i] ? "compatible" : "incompatible";
      assert(
        el.children("div").attr("class").includes(check),
        `expected policy to be ${platforms[i]} ${compatibility}`
      );
    };

    const testSelections = (
      el: JQuery<HTMLElement>,
      i: number,
      expected: boolean[]
    ) => {
      assert(
        el.prop("checked") === expected[i],
        `expected ${platforms[i]} to be ${
          expected[i] ? "selected " : "not selected"
        }`
      );
    };

    it("checks sql statement for platform compatibility", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.getAttached(".add-policy-modal").within(() => {
        cy.findByRole("button", { name: /create your own policy/i }).click();
      });

      cy.getAttached(".platform").each((el, i) => {
        testCompatibility(el, i, [true, true, true]);
      });

      // Query with unknown table name displays error message
      cy.getAttached(".policy-page__form .ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall}SELECT 1 FROM foo WHERE start_time > 1;");
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform-compatibility").within(() => {
        cy.findByText(
          "No platforms (check your query for invalid tables or tables that are supported on different platforms)"
        ).should("exist");
      });

      // Query with syntax error displays error message
      cy.getAttached(".policy-page__form .ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall}SELEC 1 FRO osquery_info WHER start_time > 1;");
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform-compatibility").within(() => {
        cy.findByText(
          "No platforms (check your query for a possible syntax error)"
        ).should("exist");
      });

      // Query with no tables treated as compatible with all platforms
      cy.getAttached(".policy-page__form .ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall}SELECT * WHERE 1 = 1;");
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform").each((el, i) => {
        testCompatibility(el, i, [true, true, true]);
      });

      // Tables defined in common table expression not factored into compatibility check
      cy.getAttached(".policy-page__form .ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall} ")
        .type(
          `WITH target_jars AS ( SELECT DISTINCT path FROM ( WITH split(word, str) AS( SELECT '', cmdline || ' ' FROM processes UNION ALL SELECT substr(str, 0, instr(str, ' ')), substr(str, instr(str, ' ') + 1) FROM split WHERE str != '') SELECT word AS path FROM split WHERE word LIKE '%.jar' UNION ALL SELECT path FROM process_open_files WHERE path LIKE '%.jar' ) ) SELECT path, matches FROM yara WHERE path IN (SELECT path FROM target_jars) AND count > 0 AND sigrule IN ( 'rule log4jJndiLookup { strings: $jndilookup = "JndiLookup" condition: $jndilookup }', 'rule log4jJavaClass { strings: $javaclass = "org/apache/logging/log4j" condition: $javaclass }' );`,
          { parseSpecialCharSequences: false }
        );
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(1000); // wait for text input debounce
      cy.getAttached(".platform").each((el, i) => {
        testCompatibility(el, i, [true, false, true]);
      });

      // Query with only macOS tables treated as compatible only with macOS
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.getAttached(" .policy-page__form .ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall} ")
        .wait(300) // wait for ace to clear text before proceeding
        .type(
          "{selectall}SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;"
        );
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform").each((el, i) => {
        testCompatibility(el, i, [true, false, false]);
      });

      // Query with macadmins extension table is not treated as incompatible
      cy.getAttached(".policy-page__form .ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall}SELECT 1 FROM mdm WHERE enrolled='true';");
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform").each((el, i) => {
        testCompatibility(el, i, [true, false, false]);
      });
    });

    it("preselects platforms to check based on platform compatiblity when saving new policy", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.getAttached(".add-policy-modal").within(() => {
        cy.findByText("Automatic login disabled (macOS)").click();
      });

      cy.getAttached(".platform-compatibility").within(() => {
        cy.getAttached(".platform").each((el, i) => {
          testCompatibility(el, i, [true, false, false]);
        });
      });
      cy.findByRole("button", { name: /save/i }).click();
      cy.getAttached(".modal__content").within(() => {
        cy.getAttached(".platform-selector").within(() => {
          cy.getAttached(".fleet-checkbox__input").each((el, i) => {
            testSelections(el, i, [true, false, false]);
          });
        });
      });
    });

    it("disables modal save button if no platforms are selected", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.getAttached(".add-policy-modal").within(() => {
        cy.findByText("Automatic login disabled (macOS)").click();
      });
      cy.findByRole("button", { name: /save/i }).click();

      cy.getAttached(".modal__content").within(() => {
        cy.getAttached(".platform-selector").within(() => {
          cy.getAttached(".fleet-checkbox__input").each((el, i) => {
            testSelections(el, i, [true, false, false]);
          });
          cy.getAttached(".fleet-checkbox__label").first().click(); // deselect macOS
          cy.getAttached(".fleet-checkbox__input").each((el, i) => {
            testSelections(el, i, [false, false, false]);
          });
        });

        cy.getAttached(".modal-cta-wrap").within(() => {
          cy.findByRole("button", { name: /save policy/i }).should(
            "be.disabled"
          );
        });
      });
    });

    it("allows user to overide preselected platforms when saving new policy", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.getAttached(".add-policy-modal").within(() => {
        cy.findByText("Automatic login disabled (macOS)").click();
      });

      cy.getAttached(".platform-compatibility").within(() => {
        cy.getAttached(".platform").each((el, i) => {
          testCompatibility(el, i, [true, false, false]);
        });
      });
      cy.findByRole("button", { name: /save/i }).click();
      cy.getAttached(".modal__content").within(() => {
        cy.getAttached(".platform-selector").within(() => {
          cy.getAttached(".fleet-checkbox__input").each((el, i) => {
            testSelections(el, i, [true, false, false]);
          });
          cy.getAttached(".fleet-checkbox__label").first().click(); // deselect macOS
          cy.getAttached(".fleet-checkbox__label").last().click(); // select Linux
          cy.getAttached(".fleet-checkbox__input").each((el, i) => {
            testSelections(el, i, [false, false, true]);
          });
        });
      });
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.findByText(/policy created/i).should("exist");

      // confirm that new policy was saved with user-selected platforms
      managePoliciesPage.visitManagePoliciesPage();
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".name__cell .button--text-link")
          .contains("Automatic login disabled (macOS)")
          .click();
      });
      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [false, false, true]);
        });
      });
    });

    it("allows user to edit existing policy platform selections", () => {
      // add a default policy for this test
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.getAttached(".add-policy-modal").within(() => {
        cy.findByText("Antivirus healthy (macOS)").click();
      });
      cy.findByRole("button", { name: /save/i }).click();
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.findByText(/policy created/i).should("exist");

      // edit platform selections for policy
      managePoliciesPage.visitManagePoliciesPage();
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".name__cell .button--text-link")
          .contains("Antivirus healthy (macOS)")
          .click();
      });
      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [true, false, false]);
        });
        cy.getAttached(".fleet-checkbox__label").first().click(); // deselect macOS
      });

      // confirm save/run buttons are disabled when no platforms are selected
      cy.findByRole("button", { name: /^Save$/ }).should("be.disabled");
      cy.findByRole("button", { name: /^Run$/ }).should("be.disabled");
      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__label").last().click(); // select Linux
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [false, false, true]);
        });
      });

      // save policy with new selection
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.findByText(/policy updated/i).should("exist");

      // confirm that policy was saved with new selection
      managePoliciesPage.visitManagePoliciesPage();
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".name__cell .button--text-link")
          .contains("Antivirus healthy (macOS)")
          .click();
      });
      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [false, false, true]);
        });
      });
    });
  });
});

describe("Policies flow (seeded)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedPolicies();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("Manage policies page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      managePoliciesPage.visitManagePoliciesPage();
    });
    it("links to manage host page filtered by policy", () => {
      // Move internal clock forward 2 hours so that policies report host results
      cy.clock(Date.now() + 1000 * 60 * 120);
      cy.getAttached(".failing_host_count__cell")
        .first()
        .within(() => {
          cy.getAttached(".button--text-link").click();
        });
      // confirm policy functionality on manage host page
      cy.getAttached(".manage-hosts__labels-active-filter-wrap").within(() => {
        cy.findByText(/filevault enabled/i).should("exist");
        cy.findByText(/no/i).should("exist").click();
        cy.findByText(/yes/i).should("exist");
        cy.get('img[alt="Remove filter"]').click();
        cy.findByText(/filevault enabled'/i).should("not.exist");
      });
    });
    it("edits an existing policy", () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".name__cell .button--text-link").first().click();
      });
      cy.getAttached(".policy-page__form .ace_scroller")
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;"
        );
      cy.getAttached(".fleet-checkbox__label").first().click();
      cy.findByRole("button", { name: /save/i }).click();
      cy.findByText(/policy updated/i).should("exist");
      cy.visit("policies/1");
      cy.getAttached(".fleet-checkbox__input").first().should("not.be.checked");
    });

    it("deletes an existing policy", () => {
      managePoliciesPage.allowsDeletePolicy();
      managePoliciesPage.verifiesDeletedPolicy();
    });
    it("creates a failing policies webhook", () => {
      cy.getAttached(".button-wrap").within(() => {
        cy.findByRole("button", { name: /manage automations/i }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        // Ensure clicking on slider after modal animation
        cy.wait(300); // eslint-disable-line cypress/no-unnecessary-waiting
        cy.getAttached(".fleet-slider").click();
        cy.getAttached(".fleet-checkbox__input").check({ force: true });
      });
      cy.getAttached("#webhook-url").click().type("http://www.foo.com/bar");
      cy.findByRole("button", { name: /^Save$/ }).click();
      // Ensure update
      cy.wait(1000); // eslint-disable-line cypress/no-unnecessary-waiting
      // Confirm failing policies webhook was added successfully
      cy.findByText(/updated policy automations/i).should("exist");
      cy.getAttached(".button-wrap").within(() => {
        cy.findByRole("button", { name: /manage automations/i }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-checkbox__input").should("be.checked");
      });
      // reset slider for subsequent tests
      cy.getAttached(".manage-automations-modal").within(() => {
        // Ensure clicking on slider after modal animation
        cy.wait(300); // eslint-disable-line cypress/no-unnecessary-waiting
        cy.getAttached(".fleet-slider").click();
      });
      cy.findByRole("button", { name: /^Save$/ }).click();
      // Confirm failing policies webhook was updated successfully
      cy.findByText(/updated policy automations/i).should("exist");
    });
    it("empty automation state prompts to create an integration", () => {
      cy.getAttached(".button-wrap").within(() => {
        cy.findByRole("button", { name: /manage automations/i }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-slider").click();
        cy.getAttached(".fleet-checkbox__input").check({ force: true });
      });
      cy.getAttached("#ticket-radio-btn").next().click();

      cy.findByText(/you have no integrations/i).should("exist");
      cy.getAttached(".manage-automations-modal__add-integration-link").click();
      // should be redirected to integrations settings page
      cy.getAttached(".table-container").within(() => {
        cy.findByText(/set up integration/i).should("exist");
      });
    });
  });
  describe("Manage policies page (mock integrations)", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.viewport(1600, 900);
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        CONFIG_INTEGRATIONS_AUTOMATIONS
      ).as("getIntegrations");
      managePoliciesPage.visitManagePoliciesPage();
      cy.wait("@getIntegrations").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
    });
    it("creates jira integration failing policies automation", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
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
        enableJiraPoliciesIntegration
      ).as("enableJiraPoliciesIntegration");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        enableJiraPoliciesIntegration
      ).as("enabledJiraPoliciesIntegration");
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.wait("@enableJiraPoliciesIntegration").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      // Confirm jira integration was added successfully
      cy.findByText(/updated policy automations/i).should("exist");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        enableJiraPoliciesIntegration
      ).as("getIntegrations");
      managePoliciesPage.visitManagePoliciesPage();
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
    it("creates zendesk integration failing policies automation", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
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
        enableZendeskPoliciesIntegration
      ).as("enableZendeskPoliciesIntegration");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        enableZendeskPoliciesIntegration
      ).as("enabledZendeskPoliciesIntegration");
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.wait("@enableZendeskPoliciesIntegration").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      // Confirm zendesk integration was added successfully
      cy.findByText(/updated policy automations/i).should("exist");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        enableZendeskPoliciesIntegration
      ).as("getIntegrations");
      managePoliciesPage.visitManagePoliciesPage();
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
    it("disables failing policies automation", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
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
      ).as("disablePoliciesAutomations");
      cy.intercept(
        "GET",
        "/api/latest/fleet/config",
        CONFIG_INTEGRATIONS_AUTOMATIONS_DISABLED
      ).as("disabledAutomations");
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.wait("@disablePoliciesAutomations").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      cy.wait("@disabledAutomations").then((configStub) => {
        console.log(JSON.stringify(configStub));
      });
      // Confirm integration was disabled successfully
      cy.findByText(/updated policy automations/i).should("exist");
      cy.getAttached(".button-wrap").within(() => {
        cy.findByRole("button", {
          name: /manage automations/i,
        }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.findByText(/policy automations disabled/i).should("exist");
      });
    });
  });
  describe("Platform compatibility", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/policies/manage");
    });
    const platforms = ["macOS", "Windows", "Linux"];

    const testSelections = (
      el: JQuery<HTMLElement>,
      i: number,
      expected: boolean[]
    ) => {
      assert(
        el.prop("checked") === expected[i],
        `expected ${platforms[i]} to be ${
          expected[i] ? "selected " : "not selected"
        }`
      );
    };
    it('preselects all platforms if API response contains `platform: ""`', () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".name__cell .button--text-link")
          .contains("Is Ubuntu, version 16.4.0 or later, installed?")
          .click();
      });
      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [true, true, true]);
        });
      });
    });
  });
});

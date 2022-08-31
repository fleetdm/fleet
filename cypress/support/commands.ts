import "@testing-library/cypress/add-commands";
import "cypress-wait-until";
import CONSTANTS from "./constants";

const { GOOD_PASSWORD } = CONSTANTS;

// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add("login", (email, password) => { ... })
//
//
// -- This is a child command --
// Cypress.Commands.add("drag", { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add("dismiss", { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite("visit", (originalFn, url, options) => { ... })

const SHELL = Cypress.platform === "win32" ? "cmd" : "bash";

Cypress.Commands.add("setup", () => {
  cy.exec("make e2e-reset-db e2e-setup", {
    timeout: 20000,
    env: { SHELL },
  });
});

Cypress.Commands.add("setupWithSoftware", () => {
  cy.exec("make e2e-reset-db e2e-setup-with-software", {
    timeout: 20000,
    env: { SHELL },
  });
});

Cypress.Commands.add("login", (email, password) => {
  email ||= "admin@example.com";
  password ||= GOOD_PASSWORD;
  cy.request("POST", "/api/latest/fleet/login", { email, password }).then(
    (resp) => {
      window.localStorage.setItem("FLEET::auth_token", resp.body.token);
    }
  );
});

Cypress.Commands.add("loginWithCySession", (email, password) => {
  email ||= "admin@example.com";
  password ||= GOOD_PASSWORD;
  cy.session([email, password], () => {
    cy.request("POST", "/api/latest/fleet/login", { email, password }).then(
      (resp) => {
        window.localStorage.setItem("FLEET::auth_token", resp.body.token);
      }
    );
  });
});

Cypress.Commands.add("logout", () => {
  cy.request({
    url: "/api/latest/fleet/logout",
    method: "POST",
    body: {},
    auth: {
      bearer: window.localStorage.getItem("FLEET::auth_token"),
    },
  }).then(() => {
    window.localStorage.removeItem("FLEET::auth_token");
  });
});

Cypress.Commands.add("setDesktopToken", (hostId, token) => {
  cy.exec(`make e2e-set-desktop-token host_id=${hostId} token=${token}`, {
    timeout: 20000,
    env: { SHELL },
  });
});

Cypress.Commands.add("seedQueries", () => {
  const queries = [
    {
      name: "Detect presence of authorized SSH keys",
      query:
        "SELECT username, authorized_keys. * FROM users CROSS JOIN authorized_keys USING (uid)",
      description:
        "Presence of authorized SSH keys may be unusual on laptops. Could be completely normal on servers, but may be worth auditing for unusual keys and/or changes.",
      observer_can_run: true,
    },
    {
      name: "Get authorized keys for Domain Joined Accounts",
      query:
        "SELECT * FROM users CROSS JOIN authorized_keys USING(uid) WHERE username IN (SELECT distinct(username) FROM last);",
      description: "List authorized_keys for each user on the system.",
      observer_can_run: false,
    },
    {
      name: "Get local user accounts",
      query:
        "SELECT uid, gid, username, description,directory, shell FROM users;",
      description:
        "Local user accounts (including domain accounts that have logged on locally (Windows)).",
      observer_can_run: false,
    },
  ];

  queries.forEach((queryForm) => {
    const { name, query, description, observer_can_run } = queryForm;
    cy.request({
      url: "/api/latest/fleet/queries",
      method: "POST",
      body: { name, query, description, observer_can_run },
      auth: {
        bearer: window.localStorage.getItem("FLEET::auth_token"),
      },
    });
  });
});

Cypress.Commands.add("seedSchedule", () => {
  const scheduledQueries = [
    {
      interval: 86400,
      platform: "",
      query_id: 1,
      removed: false,
      shard: null,
      snapshot: true,
      version: "",
    },
    {
      interval: 604800,
      platform: "linux",
      query_id: 2,
      removed: true,
      shard: 50,
      snapshot: false,
      version: "4.6.0",
    },
  ];

  scheduledQueries.forEach((scheduleForm) => {
    const {
      interval,
      platform,
      query_id,
      removed,
      shard,
      snapshot,
      version,
    } = scheduleForm;
    cy.request({
      url: "/api/latest/fleet/schedule",
      method: "POST",
      body: { interval, platform, query_id, removed, shard, snapshot, version },
      auth: {
        bearer: window.localStorage.getItem("FLEET::auth_token"),
      },
    });
  });
});

// @ts-ignore
Cypress.Commands.add("seedPacks", () => {
  const packs = [
    {
      name: "Mac starter pack",
      description: "Run all queries weekly on Mac hosts",
    },
    {
      name: "Windows starter pack",
      description: "Run all queries weekly on Windows hosts",
    },
  ];

  packs.forEach((packForm) => {
    const { name, description } = packForm;
    cy.request({
      url: "/api/latest/fleet/packs",
      method: "POST",
      body: { name, description, host_ids: [], label_ids: [], team_ids: [] },
      auth: {
        bearer: window.localStorage.getItem("FLEET::auth_token"),
      },
    });
  });
});

Cypress.Commands.add("seedPolicies", (team = "") => {
  const policies = [
    {
      name: "Is Filevault enabled on macOS devices?",
      query:
        "SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT '' AND filevault_status = 'on' LIMIT 1",
      description:
        "Checks to make sure that the Filevault feature is enabled on macOS devices.",
      resolution:
        "Choose Apple menu > System Preferences, then click Security & Privacy. Click the FileVault tab. Click the Lock icon, then enter an administrator name and password. Click Turn On FileVault.",
      platform: "darwin,linux",
    },
    {
      name: "Is Ubuntu, version 20.4.0 installed?",
      query:
        "SELECT 1 from os_version WHERE name = 'Ubuntu' AND major || '.' || minor || '.' || patch = '20.4.0';",
      description:
        "Returns yes or no for detecting operating system and version",
      resolution: "Update OS if needed",
      platform: "darwin,windows,linux",
    },
    {
      name: "Is Ubuntu, version 16.4.0 or later, installed?",
      query:
        "SELECT 1 from os_version WHERE name = 'Ubuntu' AND major || '.' || minor || '.' || patch >= '16.4.0';",
      description:
        "Returns yes or no for detecting operating system and version",
      resolution: "Update OS if needed",
      platform: "",
    },
  ];

  if (team === "apples") {
    policies.forEach((policyForm) => {
      cy.request({
        url: "/api/latest/fleet/teams/1/policies",
        method: "POST",
        body: { ...policyForm },
        auth: {
          bearer: window.localStorage.getItem("FLEET::auth_token"),
        },
      });
    });
  } else {
    policies.forEach((policyForm) => {
      cy.request({
        url: "/api/latest/fleet/policies",
        method: "POST",
        body: { ...policyForm },
        auth: {
          bearer: window.localStorage.getItem("FLEET::auth_token"),
        },
      });
    });
  }
});

Cypress.Commands.add("setupSMTP", () => {
  const body = {
    smtp_settings: {
      authentication_type: "authtype_none",
      enable_smtp: true,
      port: 1025,
      sender_address: "fleet@example.com",
      server: "localhost",
    },
  };

  cy.request({
    url: "/api/latest/fleet/config",
    method: "PATCH",
    body,
    auth: {
      bearer: window.localStorage.getItem("FLEET::auth_token"),
    },
  });
});

Cypress.Commands.add("setupSSO", (options) => {
  const body = {
    sso_settings: Object.assign(
      {},
      {
        enable_sso: true,
        enable_sso_idp_login: false,
        enable_jit_provisioning: false,
        entity_id: "https://localhost:8080",
        idp_name: "SimpleSAML",
        issuer_uri: "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
        metadata_url: "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
      },
      options
    ),
  };

  cy.request({
    url: "/api/latest/fleet/config",
    method: "PATCH",
    body,
    auth: {
      bearer: window.localStorage.getItem("FLEET::auth_token"),
    },
  });
});

Cypress.Commands.add(
  "loginSSO",
  ({ username = "sso_user", password = "user123#" } = {}) => {
    // Note these requests set cookies that are required for the SSO flow to
    // work properly. This is handled automatically by the browser.
    cy.request({
      method: "GET",
      url:
        "http://localhost:9080/simplesaml/saml2/idp/SSOService.php?spentityid=https://localhost:8080",
      followRedirect: false,
    }).then((firstResponse) => {
      const redirect = firstResponse.headers.location as string;

      cy.request({
        method: "GET",
        url: redirect,
        followRedirect: false,
      }).then((secondResponse) => {
        const el = document.createElement("html");
        el.innerHTML = secondResponse.body;
        const authState = el
          .getElementsByTagName("input")
          .namedItem("AuthState").defaultValue;

        cy.request({
          method: "POST",
          url: redirect,
          body: `username=${username}&password=${password}&AuthState=${authState}`,
          form: true,
          followRedirect: false,
        }).then((finalResponse) => {
          el.innerHTML = finalResponse.body;
          const saml = el
            .getElementsByTagName("input")
            .namedItem("SAMLResponse").defaultValue;

          // Load the callback URL with the response from the IdP
          cy.visit({
            url: "/api/v1/fleet/sso/callback",
            method: "POST",
            body: {
              SAMLResponse: saml,
            },
          });
        });
      });
    });
  }
);

Cypress.Commands.add("getEmails", () => {
  return cy
    .request("http://localhost:8025/api/v2/messages")
    .then((response) => {
      expect(response.status).to.eq(200);
      return response;
    });
});

Cypress.Commands.add("seedFree", () => {
  const authToken = window.localStorage.getItem("FLEET::auth_token");
  cy.exec("bash ./tools/api/fleet/teams/create_free", {
    env: {
      TOKEN: authToken,
      CURL_FLAGS: "-k",
      SERVER_URL: Cypress.config().baseUrl,
      // clear any value for FLEET_ENV_PATH since we set the environment explicitly just above
      FLEET_ENV_PATH: "",
      SHELL,
    },
  });
});

Cypress.Commands.add("seedPremium", () => {
  const authToken = window.localStorage.getItem("FLEET::auth_token");
  cy.exec("bash ./tools/api/fleet/teams/create_premium", {
    env: {
      TOKEN: authToken,
      CURL_FLAGS: "-k",
      SERVER_URL: Cypress.config().baseUrl,
      // clear any value for FLEET_ENV_PATH since we set the environment explicitly just above
      FLEET_ENV_PATH: "",
      SHELL,
    },
  });
});

Cypress.Commands.add("seedFigma", () => {
  const authToken = window.localStorage.getItem("FLEET::auth_token");
  cy.exec("bash ./tools/api/fleet/teams/create_figma", {
    env: {
      TOKEN: authToken,
      CURL_FLAGS: "-k",
      SERVER_URL: Cypress.config().baseUrl,
      // clear any value for FLEET_ENV_PATH since we set the environment explicitly just above
      FLEET_ENV_PATH: "",
      SHELL,
    },
  });
});

Cypress.Commands.add("addUser", (options = {}) => {
  let { password, email, globalRole } = options;
  password ||= GOOD_PASSWORD;
  email ||= `admin@example.com`;
  globalRole ||= "admin";

  cy.exec(
    `./build/fleetctl user create --context e2e --password "${password}" --email "${email}" --global-role "${globalRole}"`,
    {
      timeout: 5000,
      env: { SHELL },
    }
  );
});

// Ability to add a docker host to a team using args if ran after seedPremium()
Cypress.Commands.add("addDockerHost", (team = "") => {
  const serverPort = new URL(Cypress.config().baseUrl).port;
  // Get enroll secret
  let enrollSecretURL = "/api/latest/fleet/spec/enroll_secret";
  if (team === "apples") {
    enrollSecretURL = "/api/latest/fleet/teams/1/secrets";
  } else if (team === "oranges") {
    enrollSecretURL = "/api/latest/fleet/teams/2/secrets";
  }

  cy.request({
    url: enrollSecretURL,
    auth: {
      bearer: window.localStorage.getItem("FLEET::auth_token"),
    },
  }).then(({ body }) => {
    const enrollSecret =
      team === "" ? body.spec.secrets[0].secret : body.secrets[0].secret;

    // Start up docker-compose with enroll secret
    cy.exec(
      "docker-compose -f tools/osquery/docker-compose.yml up -d ubuntu20-osquery",
      {
        env: {
          ENROLL_SECRET: enrollSecret,
          FLEET_SERVER: `host.docker.internal:${serverPort}`,
          SHELL,
        },
      }
    );
  });
});

Cypress.Commands.add("stopDockerHost", () => {
  // Start up docker-compose with enroll secret
  cy.exec("docker-compose -f tools/osquery/docker-compose.yml stop", {
    env: {
      // Not that ENROLL_SECRET must be specified or docker-compose errors,
      // even when just trying to shut down the hosts.
      ENROLL_SECRET: "invalid",
      SHELL,
    },
  });
});

Cypress.Commands.add("clearDownloads", () => {
  // windows has issue with downloads location
  if (Cypress.platform !== "win32") {
    cy.exec(`rm -rf ${Cypress.config("downloadsFolder")}`, { env: { SHELL } });
  }
});

Cypress.Commands.add("getAttached", (selector) => {
  const uniqueAlias = `element_${selector}`;

  return cy
    .waitUntil(
      () =>
        // eslint-disable-next-line cypress/no-unnecessary-waiting
        cy
          .get(selector)
          .as(uniqueAlias)
          .wait(1)
          .then(($el) => Cypress.dom.isAttached($el)),
      { timeout: 1000, interval: 10 }
    )
    .get(`@${uniqueAlias}`);
});

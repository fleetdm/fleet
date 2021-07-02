import "@testing-library/cypress/add-commands";
import "cypress-wait-until";

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

Cypress.Commands.add("setup", () => {
  cy.exec("make e2e-reset-db e2e-setup", { timeout: 20000 });
});

Cypress.Commands.add("login", (email, password) => {
  email ||= "admin@example.com";
  password ||= "user123#";
  cy.request("POST", "/api/v1/fleet/login", { email, password }).then(
    (resp) => {
      window.localStorage.setItem("FLEET::auth_token", resp.body.token);
    }
  );
});

Cypress.Commands.add("logout", () => {
  cy.request({
    url: "/api/v1/fleet/logout",
    method: "POST",
    body: {},
    auth: {
      bearer: window.localStorage.getItem("FLEET::auth_token"),
    },
  }).then(() => {
    window.localStorage.removeItem("FLEET::auth_token");
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
  ];

  queries.forEach((queryForm) => {
    const { name, query, description, observer_can_run } = queryForm;
    cy.request({
      url: "/api/v1/fleet/queries",
      method: "POST",
      body: { name, query, description, observer_can_run },
      auth: {
        bearer: window.localStorage.getItem("FLEET::auth_token"),
      },
    });
  });
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
    url: "/api/v1/fleet/config",
    method: "PATCH",
    body,
    auth: {
      bearer: window.localStorage.getItem("FLEET::auth_token"),
    },
  });
});

Cypress.Commands.add("setupSSO", (enable_idp_login = false) => {
  const body = {
    sso_settings: {
      enable_sso: true,
      enable_sso_idp_login: enable_idp_login,
      entity_id: "https://localhost:8080",
      idp_name: "SimpleSAML",
      issuer_uri: "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
      metadata_url: "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
    },
  };

  cy.request({
    url: "/api/v1/fleet/config",
    method: "PATCH",
    body,
    auth: {
      bearer: window.localStorage.getItem("FLEET::auth_token"),
    },
  });
});

Cypress.Commands.add("loginSSO", () => {
  // Note these requests set cookies that are required for the SSO flow to
  // work properly. This is handled automatically by the browser.
  cy.request({
    method: "GET",
    url:
      "http://localhost:9080/simplesaml/saml2/idp/SSOService.php?spentityid=https://localhost:8080",
    followRedirect: false,
  }).then((firstResponse) => {
    const redirect = firstResponse.headers.location;

    cy.request({
      method: "GET",
      url: redirect,
      followRedirect: false,
    }).then((secondResponse) => {
      const el = document.createElement("html");
      el.innerHTML = secondResponse.body;
      const authState = el.getElementsByTagName("input").namedItem("AuthState")
        .defaultValue;

      cy.request({
        method: "POST",
        url: redirect,
        body: `username=sso_user&password=user123#&AuthState=${authState}`,
        form: true,
        followRedirect: false,
      }).then((finalResponse) => {
        el.innerHTML = finalResponse.body;
        const saml = el.getElementsByTagName("input").namedItem("SAMLResponse")
          .defaultValue;

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
});

Cypress.Commands.add("getEmails", () => {
  return cy
    .request("http://localhost:8025/api/v2/messages")
    .then((response) => {
      expect(response.status).to.eq(200);
      return response;
    });
});

Cypress.Commands.add("seedCore", () => {
  const authToken = window.localStorage.getItem("FLEET::auth_token");
  cy.exec("bash ./tools/api/fleet/teams/create_core", {
    env: {
      TOKEN: authToken,
      CURL_FLAGS: "-k",
      SERVER_URL: Cypress.config().baseUrl,
      // clear any value for FLEET_ENV_PATH since we set the environment explicitly just above
      FLEET_ENV_PATH: "",
    },
  });
});

Cypress.Commands.add("seedBasic", () => {
  const authToken = window.localStorage.getItem("FLEET::auth_token");
  cy.exec("bash ./tools/api/fleet/teams/create_basic", {
    env: {
      TOKEN: authToken,
      CURL_FLAGS: "-k",
      SERVER_URL: Cypress.config().baseUrl,
      // clear any value for FLEET_ENV_PATH since we set the environment explicitly just above
      FLEET_ENV_PATH: "",
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
    },
  });
});

Cypress.Commands.add("addUser", (options = {}) => {
  let { password, email, globalRole } = options;
  password ||= "test123#";
  email ||= `admin@example.com`;
  globalRole ||= "admin";

  cy.exec(
    `./build/fleetctl user create --context e2e --password "${password}" --email "${email}" --global-role "${globalRole}"`,
    { timeout: 5000 }
  );
});

// Ability to add a docker host to a team using args if ran after seedBasic()
Cypress.Commands.add("addDockerHost", (team = "") => {
  const serverPort = new URL(Cypress.config().baseUrl).port;
  // Get enroll secret
  let enrollSecretURL = "/api/v1/fleet/spec/enroll_secret";
  if (team === "apples") {
    enrollSecretURL = "/api/v1/fleet/teams/1/secrets";
  } else if (team === "oranges") {
    enrollSecretURL = "/api/v1/fleet/teams/2/secrets";
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
    },
  });
});

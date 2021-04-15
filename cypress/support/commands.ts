import "@testing-library/cypress/add-commands";

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

Cypress.Commands.add("login", (username, password) => {
  username ||= "test";
  password ||= "admin123#";
  cy.request("POST", "/api/v1/fleet/login", { username, password }).then(
    (resp) => {
      window.localStorage.setItem("KOLIDE::auth_token", resp.body.token);
    }
  );
});

Cypress.Commands.add("logout", () => {
  cy.request({
    url: "/api/v1/fleet/logout",
    method: "POST",
    body: {},
    auth: {
      bearer: window.localStorage.getItem("KOLIDE::auth_token"),
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
      bearer: window.localStorage.getItem("KOLIDE::auth_token"),
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
        body: `username=user1&password=user1pass&AuthState=${authState}`,
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

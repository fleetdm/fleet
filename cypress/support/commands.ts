import '@testing-library/cypress/add-commands';

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

Cypress.Commands.add('setup', () => {
  cy.exec('make e2e-reset-db e2e-setup', { timeout: 10000 });
});

Cypress.Commands.add('setupSMTP', () => {
  const body = {
    smtp_settings: {
      authentication_type: 'authtype_none',
      enable_smtp: true,
      port: 1025,
      sender_address: 'gabriel+dev@fleetdm.com',
      server: 'localhost',
    },
  };

  cy.request({
    url: '/api/v1/fleet/config',
    method: 'PATCH',
    body,
    auth: {
      bearer: window.localStorage.getItem('KOLIDE::auth_token'),
    },
  });
});

Cypress.Commands.add('login', (username, password) => {
  username ||= 'test';
  password ||= 'admin123#';
  cy.request('POST', '/api/v1/fleet/login', { username, password })
    .then((resp) => {
      window.localStorage.setItem('KOLIDE::auth_token', resp.body.token);
    });
});

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
  cy.exec('make e2e-reset-db e2e-setup', { timeout: 5000 });
});

Cypress.Commands.add('login', (username, password) => {
  username ||= 'test';
  password ||= 'admin123#';
  cy.request('POST', '/api/v1/fleet/login', { username, password })
    .then((resp) => {
      window.localStorage.setItem('KOLIDE::auth_token', resp.body.token);
    });
});

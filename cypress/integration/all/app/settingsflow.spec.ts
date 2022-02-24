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

  describe("Teams settings page", () => {
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
        .click()
        .blur();

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

      cy.findByLabelText(/metadata url/i).should(
        "have.value",
        "http://github.com/fleetdm/fleet"
      );

      cy.findByLabelText(/sender address/i).should(
        "have.value",
        "rachel@example.com"
      );

      cy.findByLabelText(/smtp server/i).should("have.value", "localhost");

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
});

describe("Settings flow", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
  });

  // We're using `scrollBehavior: 'center'` as a default
  // because the sticky header blocks the elements.
  it(
    "Modifies and updates all setting successfully",
    { scrollBehavior: "center" },
    () => {
      cy.visit("/settings/organization");

      cy.findByLabelText(/organization name/i)
        .click()
        .type("{selectall}{backspace}TJ's Run");

      cy.findByLabelText(/organization avatar url/i)
        .click()
        .type("http://tjsrun.com/img/logo.png");

      cy.findByLabelText(/fleet app url/i)
        .click()
        .type("{selectall}{backspace}https://localhost:5000");

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

      // only allowed to fill in either metadata || metadata url
      cy.findByLabelText(/metadata url/i)
        .click()
        .type("http://github.com/fleetdm/fleet");

      cy.findByLabelText(/allow sso login initiated/i).check({ force: true });

      cy.findByLabelText(/enable smtp/i).check({ force: true });

      cy.findByLabelText(/sender address/i)
        .click()
        .type("rachel@example.com");

      cy.findByLabelText(/smtp server/i)
        .click()
        .type("localhost");

      cy.get("#port").click().type("{selectall}{backspace}1025");

      cy.findByLabelText(/use ssl\/tls/i).check({ force: true });

      cy.findByLabelText(/smtp username/i)
        .click()
        .type("rachelsusername");

      cy.findByLabelText(/smtp password/i)
        .click()
        .type("rachelspassword");

      cy.findByLabelText(/enable host status webhook/i).check({ force: true });

      cy.findByLabelText(/destination url/i)
        .click()
        .type("http://server.com/example");

      cy.get(".app-config-form__host-percentage").click();

      cy.get(".app-config-form__host-percentage").contains(/5%/i).click();

      cy.get(".app-config-form__days-count").click();

      cy.get(".app-config-form__days-count")
        .contains(/7 days/i)
        .click();

      cy.findByLabelText(/domain/i)
        .click()
        .type("http://www.fleetdm.com");

      // can't grab button from the label because the button is a child element and doesn't have a for attribute
      // couldn't figure out how to write a for attribute on kolide button
      // Repeated Error Message: Timed out retrying after 4000ms: Found a label with the text of: /verify ssl certs/i, however no form control was found associated to that label. Make sure you're using the "for" attribute or "aria-labelledby" attribute correctly.
      cy.findByLabelText(/verify ssl certs/i).check({ force: true });
      cy.findByLabelText(/enable starttls/i).check({ force: true });
      cy.findByLabelText(/^host expiry$/i).check({ force: true });

      cy.findByLabelText(/host expiry window/i)
        .click()
        .type("{selectall}{backspace}5");

      cy.findByLabelText(/disable live queries/i).check({ force: true });

      // Update settings
      cy.findByRole("button", { name: /update settings/i }).click();

      cy.findByText(/settings updated/i).should("exist");

      cy.visit("/settings/organization");

      cy.findByLabelText(/organization name/i).should("have.value", "TJ's Run");

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

      cy.get("#port").should("have.value", "1025");

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
    }
  );
});

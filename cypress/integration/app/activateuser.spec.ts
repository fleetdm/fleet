describe("Invite user", () => {
  it("Enable SMTP setting", () => {
    cy.setup();
    cy.login();

    cy.visit("/settings/organization");

    cy.findByLabelText(/enable smtp/i).check({ force: true });

    cy.findByLabelText(/sender address/i)
      .click()
      .type("rachel@fleetdm.com");

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

    cy.findByRole("button", { name: /update settings/i }).click();
  });

  it("Invite user", () => {
    cy.findByRole("tab", { name: /^users$/i }).click();

    cy.contains("button", /invite user/i).click();

    cy.findByLabelText(/name/i).click().type("Ash Ketchum");

    cy.findByLabelText(/email/i).click().type("ash@fleetdm.com");

    cy.findByRole("button", { name: /invite/i }).click();

    cy.wait(3000);
  });

  let inviteLink;

  it("Accept invite and create user", () => {
    const regex = /\/login\/invites\/[a-zA-Z0-9=?%&@._-]*/gm;

    cy.getEmails().then((response) => {
      expect(response.body.items[0].To[0]).to.have.property("Domain");
      expect(response.body.items[0].To[0].Mailbox).to.equal("ash");
      expect(response.body.items[0].To[0].Domain).to.equal("fleetdm.com");
      expect(response.body.items[0].From.Mailbox).to.equal("rachel");
      expect(response.body.items[0].From.Domain).to.equal("fleetdm.com");
      inviteLink = response.body.items[0].Content.Body.match(regex);
    });
  });

  it("Activate new user", () => {
    cy.visit(inviteLink[0]);

    cy.findByLabelText(/username/i)
      .click()
      .type("ash.ketchum");

    // ^$ exact match
    cy.findByLabelText(/^password$/i)
      .click()
      .type("#pikachu1");

    cy.findByLabelText(/confirm password/i)
      .click()
      .type("#pikachu1");

    cy.findByRole("button", { name: /submit/i }).click();

    cy.visit("/login");

    cy.findByLabelText(/username/i)
      .click()
      .type("test");

    cy.findByLabelText(/password/i)
      .click()
      .type("admin123#");

    cy.findByRole("button", { name: /login/i }).click();

    cy.wait(3000);

    cy.visit("/settings/organization");

    cy.findByRole("tab", { name: /^users$/i }).click();

    cy.get("tbody>tr")
      .contains("ash.ketchum")
      .next()
      .findByText(/active/i)
      .should("exist");
  });
});

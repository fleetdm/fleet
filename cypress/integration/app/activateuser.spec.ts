describe("User invite and activation", () => {
  let inviteLink = {};

  it("Invites and activates a user", () => {
    cy.setup();
    cy.login();
    cy.setupSMTP();

    cy.visit("/settings/organization");

    cy.findByRole("tab", { name: /^users$/i }).click();

    cy.contains("button", /invite user/i).click();

    cy.findByLabelText(/name/i).click().type("Ash Ketchum");

    cy.findByLabelText(/email/i).click().type("ash@fleetdm.com");

    cy.findByRole("button", { name: /invite/i }).click();

    cy.wait(6000); // eslint-disable-line cypress/no-unnecessary-waiting

    const regex = /\/login\/invites\/[a-zA-Z0-9=?%&@._-]*/gm;

    cy.getEmails().then((response) => {
      expect(response.body.items[0].To[0]).to.have.property("Domain");
      expect(response.body.items[0].To[0].Mailbox).to.equal("ash");
      expect(response.body.items[0].To[0].Domain).to.equal("fleetdm.com");
      expect(response.body.items[0].From.Mailbox).to.equal("gabriel+dev");
      expect(response.body.items[0].From.Domain).to.equal("fleetdm.com");
      const match = response.body.items[0].Content.Body.match(regex);
      inviteLink["url"] = match[0];
    });
  });

  it("activate user", () => {
    cy.visit(inviteLink);

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

    cy.wait(3000); // eslint-disable-line cypress/no-unnecessary-waiting

    cy.logout();
    cy.login();

    cy.visit("/settings/organization");

    cy.findByRole("tab", { name: /^users$/i }).click();

    cy.get("tbody>tr")
      .contains("ash.ketchum")
      .next()
      .findByText(/active/i)
      .should("exist");
  });
});

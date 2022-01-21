describe("User invite and activation", () => {
  beforeEach(() => {
    cy.setup();
    cy.login();
    cy.setupSMTP();
  });

  it("Invite and activate a user successfully", () => {
    // Invite user
    cy.visit("/settings/organization");

    cy.getAttached(".component__tabs-wrapper").within(() => {
      cy.findByRole("tab", { name: /^users$/i }).click();
    });

    cy.getAttached(".user-management").within(() => {
      cy.contains("button", /create user/i).click();
    });

    cy.getAttached(".create-user-modal").within(() => {
      cy.findByLabelText(/name/i).click().type("Ash Ketchum");

      cy.findByLabelText(/email/i).click().type("ash@example.com");

      cy.getAttached(".create-user-form__new-user-radios").within(() => {
        cy.findByRole("radio", { name: "Invite user" }).parent().click();
      });

      cy.findByRole("button", { name: /^create$/i }).click();
    });

    // Ensure the email has been delivered
    cy.wait(3000); // eslint-disable-line cypress/no-unnecessary-waiting

    cy.logout();

    // Retrieve user invite in email
    const inviteLink = {};

    const regex = /\/login\/invites\/[a-zA-Z0-9=?%&@._-]*/gm;

    cy.getEmails().then((response) => {
      expect(response.body.items[0].To[0]).to.have.property("Domain");
      expect(response.body.items[0].To[0].Mailbox).to.equal("ash");
      expect(response.body.items[0].To[0].Domain).to.equal("example.com");
      expect(response.body.items[0].From.Mailbox).to.equal("fleet");
      expect(response.body.items[0].From.Domain).to.equal("example.com");
      console.log(response.body.items[0]);
      const match = response.body.items[0].Content.Body.match(regex);
      inviteLink.url = match[0];
    });

    // Activate user
    cy.visit(inviteLink);

    cy.getAttached(".confirm-invite-page").within(() => {
      cy.findByLabelText(/full name/i)
        .click()
        .type("{selectall}{backspace}Ash Ketchum");

      cy.findByLabelText(/^password$/i)
        .click()
        .type("#pikachu1");

      cy.findByLabelText(/confirm password/i)
        .click()
        .type("#pikachu1");

      cy.findByRole("button", { name: /submit/i }).click();
    });

    // View user as admin
    cy.login();

    cy.visit("/settings/organization");

    cy.getAttached(".component__tabs-wrapper").within(() => {
      cy.findByRole("tab", { name: /^users$/i }).click();
    });

    cy.getAttached("tbody>tr>td")
      .contains("Ash Ketchum")
      .parent()
      .next()
      .findByText(/active/i)
      .should("exist");
  });
});

describe("Labels flow", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("Manage hosts page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/hosts/manage");
    });
    it("creates a custom label", () => {
      cy.findByRole("button", { name: /add label/i }).click();
      cy.getAttached(".ace_content").type(
        "{selectall}{backspace}SELECT * FROM users;"
      );
      cy.findByLabelText(/name/i).click().type("Show all MAC users");
      cy.findByLabelText(/description/i)
        .click()
        .type("Select all MAC users.");
      cy.getAttached(".label-form__form-field--platform > .Select").click();
      cy.getAttached(".Select-menu-outer").within(() => {
        cy.findByText(/macOS/i).click();
      });
      cy.findByRole("button", { name: /save label/i }).click();
      cy.findByText(/label created/i).should("exist");
    });
    it("edits a custom label", () => {
      cy.getAttached(".host-side-panel").within(() => {
        cy.findByText(/show all mac users/i).click();
      });
      cy.getAttached(".manage-hosts__label-block button").first().click();
      // SQL and Platform are immutable fields
      cy.findByLabelText(/name/i).clear().type("Show all mac usernames");
      cy.findByLabelText(/description/i)
        .clear()
        .type("Select all usernames on Mac.");
      cy.findByText(/select one/i).should("not.exist");
      cy.findByRole("button", { name: /update label/i }).click();
      cy.findByText(/label updated/i).should("exist");
    });
    it("deletes a custom label", () => {
      cy.getAttached(".host-side-panel").within(() => {
        cy.findByText(/show all mac usernames/i).click();
      });
      cy.getAttached(".manage-hosts__label-block button").last().click();
      cy.getAttached(".delete-label-modal > .button--alert")
        .contains("button", /delete/i)
        .click();
      cy.getAttached(".host-side-panel").within(() => {
        cy.findByText(/show all mac usernames/i).should("not.exist");
      });
    });
    it("creates labels with special characters", () => {
      cy.findByRole("button", { name: /add label/i }).click();
      cy.getAttached(".ace_content").type(
        "{selectall}{backspace}SELECT * FROM users;"
      );
      cy.findByLabelText(/name/i)
        .click()
        .type("** Special label (Mac / Users)");
      cy.findByLabelText(/description/i)
        .click()
        .type("Select all MAC users using special characters.");
      cy.getAttached(".label-form__form-field--platform > .Select").click();
      cy.getAttached(".Select-menu-outer").within(() => {
        cy.findByText(/macOS/i).click();
      });
      cy.findByRole("button", { name: /save label/i }).click();
      cy.findByText(/label created/i).should("exist");
    });
    it("searches labels with special characters", () => {
      cy.getAttached("#tags-filter").type("{selectall}{backspace}**");
      cy.findByText(/Special label/i).should("exist");
    });
  });
});

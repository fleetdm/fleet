import manageHostsPage from "../../pages/manageHostsPage";

describe("Labels flow", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.addDockerHost();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
    cy.stopDockerHost();
  });

  describe("Manage hosts page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      manageHostsPage.visitsManageHostsPage();
    });
    it("creates a custom label", () => {
      cy.getAttached(".label-filter-select__control").click();
      cy.findByRole("button", { name: /add label/i }).click();
      cy.getAttached(".label-form__text-editor-wrapper .ace_content").type(
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
      cy.getAttached(".label-filter-select__control").click();
      cy.findByText(/Show all MAC users/i).click();
      cy.findByRole("button", { name: /edit label/i }).click();
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
      cy.getAttached(".label-filter-select__control").click();
      cy.findByText(/Show all mac usernames/i).click();
      cy.findByRole("button", { name: /delete label/i }).click();
      cy.getAttached(".delete-label-modal")
        .contains("button", /delete/i)
        .click();
      cy.getAttached(".label-filter-select__control").within(() => {
        cy.findByText(/show all mac usernames/i).should("not.exist");
      });
    });
    it("creates labels with special characters", () => {
      cy.getAttached(".label-filter-select__control").click();
      cy.findByRole("button", { name: /add label/i }).click();
      cy.getAttached(".label-form__text-editor-wrapper .ace_content").type(
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
      cy.getAttached(".label-filter-select__control").click();
      cy.findByPlaceholderText(/filter labels by name.../i).type(
        "{selectall}{backspace}**"
      );
      cy.findByText(/Special label/i).should("exist");
    });
  });
});

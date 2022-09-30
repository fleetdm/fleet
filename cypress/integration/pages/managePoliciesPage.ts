const managePoliciesPage = {
  visitManagePoliciesPage: () => {
    cy.visit("/policies/manage");
  },

  hidesButton: (text: string) => {
    cy.contains("button", text).should("not.exist");
  },

  allowsAddDefaultPolicy: () => {
    cy.findByRole("button", { name: /add a policy/i }).click();
    // Add a default policy
    cy.findByText(/gatekeeper enabled/i).click();
    cy.getAttached(".policy-form__button-wrap").within(() => {
      cy.findByRole("button", { name: /run/i }).should("exist");
      cy.findByRole("button", { name: /save/i }).click();
    });
  },

  verifiesAddedDefaultPolicy: () => {
    cy.getAttached(".modal-cta-wrap").within(() => {
      cy.findByRole("button", { name: /save policy/i }).click();
    });
    cy.findByText(/policy created/i).should("exist");
    cy.findByText(/gatekeeper enabled/i).should("exist");
  },

  allowsAutomatePolicy: () => {
    cy.getAttached(".button-wrap")
      .findByRole("button", { name: /manage automations/i })
      .click();

    cy.getAttached(".manage-automations-modal").within(() => {
      cy.getAttached(".fleet-slider").click();
      cy.getAttached(".fleet-checkbox__input").check({ force: true });
      cy.getAttached("#webhook-url").clear().type("https://example.com/admin");
    });
  },

  verifiesAutomatedPolicy: () => {
    cy.getAttached(".manage-automations-modal").within(() => {
      cy.findByRole("button", { name: /save/i }).click();
    });
    cy.findByText(/successfully updated policy automations/i).should("exist");
  },

  allowsDeletePolicy: () => {
    cy.getAttached("tbody").within(() => {
      cy.getAttached("tr")
        .first()
        .within(() => {
          cy.getAttached(".fleet-checkbox__input").check({
            force: true,
          });
        });
    });
    cy.findByRole("button", { name: /delete/i }).click();
    cy.getAttached(".delete-policy-modal").within(() => {
      cy.findByRole("button", { name: /delete/i }).should("be.enabled");
    });
  },

  verifiesDeletedPolicy: () => {
    cy.getAttached(".delete-policy-modal").within(() => {
      cy.findByRole("button", { name: /delete/i }).click();
    });
    cy.findByText(/deleted policy/i).should("exist");
    cy.findByText(/backup/i).should("not.exist");
  },

  allowsSelectRunSavePolicy: (name = "gatekeeper") => {
    cy.getAttached(".data-table__table").within(() => {
      cy.findByRole("button", { name: RegExp(name, "i") }).click();
    });
    cy.getAttached(".policy-form__button-wrap").within(() => {
      cy.findByRole("button", { name: /run/i }).should("exist");
      cy.findByRole("button", { name: /save/i }).should("exist");
    });
  },

  allowsViewPolicyOnly: () => {
    cy.getAttached("tbody").within(() => {
      cy.getAttached("tr")
        .first()
        .within(() => {
          cy.contains(".fleet-checkbox__input").should("not.exist");
          cy.findByRole("button", { name: /filevault/i }).click();
        });
    });
    cy.getAttached(".policy-form__wrapper").within(() => {
      cy.findByRole("button", { name: /run/i }).should("not.exist");
      cy.findByRole("button", { name: /save/i }).should("not.exist");
    });
  },

  allowsRunSavePolicy: () => {
    cy.getAttached(".data-table__table").within(() => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.findByRole("button", {
              name: /gatekeeper/i,
            }).click();
          });
      });
    });
    cy.getAttached(".policy-form__button-wrap").within(() => {
      cy.findByRole("button", { name: /run/i }).should("exist");
      cy.findByRole("button", { name: /save/i }).should("exist");
    });
  },
};

export default managePoliciesPage;

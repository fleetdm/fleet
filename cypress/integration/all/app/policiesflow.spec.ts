describe("Policies flow (empty)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("Manage policies page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/policies/manage");
    });
    it("creates a custom policy", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.findByText(/create your own policy/i).click();
      cy.getAttached(".ace_scroller")
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM users WHERE username = 'backup' LIMIT 1;"
        );
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.getAttached(".policy-form__policy-save-modal-name")
        .click()
        .type("Does the device have a user named 'backup'?");
      cy.getAttached(".policy-form__policy-save-modal-description")
        .click()
        .type("Returns yes or no for having a user named 'backup'");
      cy.getAttached(".policy-form__policy-save-modal-resolution")
        .click()
        .type("Create a user named 'backup'");
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.findByText(/policy created/i).should("exist");
    });

    it("creates a default policy", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.findByText(/gatekeeper enabled/i).click();
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.getAttached(".policy-form__button-wrap--modal").within(() => {
        cy.findAllByRole("button", { name: /^Save$/ }).click();
      });
      cy.findByText(/policy created/i).should("exist");
    });
  });

  describe("Platform compatibility", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/policies/manage");
    });
    const platforms = ["macOS", "Windows", "Linux"];

    const testCompatibility = (
      el: JQuery<HTMLElement>,
      i: number,
      expected: boolean[]
    ) => {
      const check = expected[i] ? "compatible" : "incompatible";
      assert(
        el.children("img").attr("alt") === check,
        `expected policy to be ${platforms[i]} ${check}`
      );
    };

    const testSelections = (
      el: JQuery<HTMLElement>,
      i: number,
      expected: boolean[]
    ) => {
      assert(
        el.prop("checked") === expected[i],
        `expected ${platforms[i]} to be ${
          expected[i] ? "selected " : "not selected"
        }`
      );
    };

    it("checks sql statement for platform compatibility", () => {
      cy.visit("/policies/manage");
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.getAttached(".add-policy-modal__modal").within(() => {
        cy.findByRole("button", { name: /create your own policy/i }).click();
      });

      cy.getAttached(".platform").each((el, i) => {
        testCompatibility(el, i, [true, true, true]);
      });

      // Query with unknown table name displays error message
      cy.getAttached(".ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall}SELECT 1 FROM foo WHERE start_time > 1;");
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform-compatibility").within(() => {
        cy.findByText(
          "No platforms (check your query for invalid tables or tables that are supported on different platforms)"
        ).should("exist");
      });

      // Query with syntax error displays error message
      cy.getAttached(".ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall}SELEC 1 FRO osquery_info WHER start_time > 1;");
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform-compatibility").within(() => {
        cy.findByText(
          "No platforms (check your query for a possible syntax error)"
        ).should("exist");
      });

      // Query with no tables treated as compatible with all platforms
      cy.getAttached(".ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall}SELECT * WHERE 1 = 1;");
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform").each((el, i) => {
        testCompatibility(el, i, [true, true, true]);
      });

      // Tables defined in common table expression not factored into compatibility check
      cy.getAttached(".ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall} ")
        .type(
          `WITH target_jars AS ( SELECT DISTINCT path FROM ( WITH split(word, str) AS( SELECT '', cmdline || ' ' FROM processes UNION ALL SELECT substr(str, 0, instr(str, ' ')), substr(str, instr(str, ' ') + 1) FROM split WHERE str != '') SELECT word AS path FROM split WHERE word LIKE '%.jar' UNION ALL SELECT path FROM process_open_files WHERE path LIKE '%.jar' ) ) SELECT path, matches FROM yara WHERE path IN (SELECT path FROM target_jars) AND count > 0 AND sigrule IN ( 'rule log4jJndiLookup { strings: $jndilookup = "JndiLookup" condition: $jndilookup }', 'rule log4jJavaClass { strings: $javaclass = "org/apache/logging/log4j" condition: $javaclass }' );`,
          { parseSpecialCharSequences: false }
        );
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform").each((el, i) => {
        testCompatibility(el, i, [true, false, true]);
      });

      // Query with only macOS tables treated as compatible only with macOS
      cy.getAttached(".ace_scroller")
        .first()
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;"
        );
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform").each((el, i) => {
        testCompatibility(el, i, [true, false, false]);
      });

      // Query with macadmins extension table is not treated as incompatible
      cy.getAttached(".ace_scroller")
        .first()
        .click({ force: true })
        .type("{selectall}SELECT 1 FROM mdm WHERE enrolled='true';");
      // eslint-disable-next-line cypress/no-unnecessary-waiting
      cy.wait(700); // wait for text input debounce
      cy.getAttached(".platform").each((el, i) => {
        testCompatibility(el, i, [true, false, false]);
      });
    });

    it("preselects platforms to check based on platform compatiblity when saving new policy", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.getAttached(".add-policy-modal__modal").within(() => {
        cy.findByText("Automatic login disabled (macOS)").click();
      });

      cy.getAttached(".platform-compatibility").within(() => {
        cy.getAttached(".platform").each((el, i) => {
          testCompatibility(el, i, [true, false, false]);
        });
      });
      cy.findByRole("button", { name: /save policy/i }).click(); // open save policy modal

      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [true, false, false]);
        });
      });
    });

    it("disables modal save button if no platforms are selected", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.getAttached(".add-policy-modal__modal").within(() => {
        cy.findByText("Automatic login disabled (macOS)").click();
      });
      cy.findByRole("button", { name: /save policy/i }).click(); // open save policy modal

      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [true, false, false]);
        });
        cy.getAttached(".fleet-checkbox__label").first().click(); // deselect macOS
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [false, false, false]);
        });
      });
      cy.findByRole("button", { name: /^Save$/ }).should("be.disabled");
    });

    it("allows user to overide preselected platforms when saving new policy", () => {
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.getAttached(".add-policy-modal__modal").within(() => {
        cy.findByText("Automatic login disabled (macOS)").click();
      });

      cy.getAttached(".platform-compatibility").within(() => {
        cy.getAttached(".platform").each((el, i) => {
          testCompatibility(el, i, [true, false, false]);
        });
      });
      cy.findByRole("button", { name: /save policy/i }).click(); // open save policy modal

      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [true, false, false]);
        });
        cy.getAttached(".fleet-checkbox__label").first().click(); // deselect macOS
        cy.getAttached(".fleet-checkbox__label").last().click(); // select Linux
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [false, false, true]);
        });
      });
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.findByText(/policy created/i).should("exist");

      // confirm that new policy was saved with user-selected platforms
      cy.visit("policies/manage");
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".name__cell .button--text-link")
          .contains("Automatic login disabled (macOS)")
          .click();
      });
      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [false, false, true]);
        });
      });
    });

    it("allows user to edit existing policy platform selections", () => {
      // add a default policy for this test
      cy.getAttached(".manage-policies-page__header-wrap").within(() => {
        cy.findByText(/add a policy/i).click();
      });
      cy.getAttached(".add-policy-modal__modal").within(() => {
        cy.findByText("Antivirus healthy (macOS)").click();
      });
      cy.findByRole("button", { name: /save policy/i }).click();
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.findByText(/policy created/i).should("exist");

      // edit platform selections for policy
      cy.visit("policies/manage");
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".name__cell .button--text-link")
          .contains("Antivirus healthy (macOS)")
          .click();
      });
      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [true, false, false]);
        });
        cy.getAttached(".fleet-checkbox__label").first().click(); // deselect macOS
      });

      // confirm save/run buttons are disabled when no platforms are selected
      cy.findByRole("button", { name: /^Save$/ }).should("be.disabled");
      cy.findByRole("button", { name: /^Run$/ }).should("be.disabled");
      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__label").last().click(); // select Linux
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [false, false, true]);
        });
      });

      // save policy with new selection
      cy.findByRole("button", { name: /^Save$/ }).click();
      cy.findByText(/policy updated/i).should("exist");

      // confirm that policy was saved with new selection
      cy.visit("policies/manage");
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".name__cell .button--text-link")
          .contains("Antivirus healthy (macOS)")
          .click();
      });
      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [false, false, true]);
        });
      });
    });
  });
});

describe("Policies flow (seeded)", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.setup();
    cy.loginWithCySession();
    cy.seedPolicies();
    cy.viewport(1200, 660);
  });
  after(() => {
    cy.logout();
  });

  describe("Manage policies page", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/policies/manage");
    });
    it("links to manage host page filtered by policy", () => {
      cy.getAttached(".failing_host_count__cell")
        .first()
        .within(() => {
          cy.getAttached(".button--text-link").click();
        });
      // confirm policy functionality on manage host page
      cy.getAttached(".manage-hosts__policies-filter-block").within(() => {
        cy.findByText(/filevault enabled/i).should("exist");
        cy.findByText(/no/i).should("exist").click();
        cy.findByText(/yes/i).should("exist");
        cy.get('img[alt="Remove policy filter"]').click();
        cy.findByText(/filevault enabled'/i).should("not.exist");
      });
    });
    it("edits an existing policy", () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".name__cell .button--text-link").first().click();
      });
      cy.getAttached(".ace_scroller")
        .click({ force: true })
        .type(
          "{selectall}SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;"
        );
      cy.getAttached(".fleet-checkbox__label").first().click();
      cy.getAttached(".policy-form__save").click();
      cy.findByText(/policy updated/i).should("exist");
      cy.visit("policies/1");
      cy.getAttached(".fleet-checkbox__input").first().should("not.be.checked");
    });

    it("deletes an existing policy", () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached("tr")
          .first()
          .within(() => {
            cy.getAttached(".fleet-checkbox__input").check({ force: true });
          });
      });
      cy.findByRole("button", { name: /delete/i }).click();
      cy.getAttached(".remove-policies-modal").within(() => {
        cy.findByRole("button", { name: /cancel/i }).should("exist");
        cy.findByRole("button", { name: /delete/i }).click();
      });
      cy.findByText(/removed policy/i).should("exist");
      cy.findByText(/backup/i).should("not.exist");
    });
    it("creates a failing policies webhook", () => {
      cy.getAttached(".button-wrap").within(() => {
        cy.findByRole("button", { name: /manage automations/i }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-slider").click();
        cy.getAttached(".fleet-checkbox__input").check({ force: true });
      });
      cy.getAttached("#webhook-url").click().type("www.foo.com/bar");
      cy.findByRole("button", { name: /^Save$/ }).click();
      // Confirm failing policies webhook was added successfully
      cy.findByText(/updated policy automations/i).should("exist");
      cy.getAttached(".button-wrap").within(() => {
        cy.findByRole("button", { name: /manage automations/i }).click();
      });
      cy.getAttached(".manage-automations-modal").within(() => {
        cy.getAttached(".fleet-checkbox__input").should("be.checked");
      });
    });
  });
  describe("Platform compatibility", () => {
    beforeEach(() => {
      cy.loginWithCySession();
      cy.visit("/policies/manage");
    });
    const platforms = ["macOS", "Windows", "Linux"];

    const testSelections = (
      el: JQuery<HTMLElement>,
      i: number,
      expected: boolean[]
    ) => {
      assert(
        el.prop("checked") === expected[i],
        `expected ${platforms[i]} to be ${
          expected[i] ? "selected " : "not selected"
        }`
      );
    };
    it('preselects all platforms if API response contains `platform: ""`', () => {
      cy.getAttached("tbody").within(() => {
        cy.getAttached(".name__cell .button--text-link")
          .contains("Is Ubuntu, version 16.4.0 or later, installed?")
          .click();
      });
      cy.getAttached(".platform-selector").within(() => {
        cy.getAttached(".fleet-checkbox__input").each((el, i) => {
          testSelections(el, i, [true, true, true]);
        });
      });
    });
  });
});

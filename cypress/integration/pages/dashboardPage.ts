const dashboardPage = {
  visitsDashboardPage: () => {
    cy.visit("/dashboard");
  },

  switchesPlatform: (platform = "") => {
    cy.getAttached(".homepage__platform_dropdown").click();
    cy.getAttached(".Select-menu-outer").within(() => {
      cy.findAllByText(platform).click();
    });
  },

  displaysCards: (platform = "", tier = "free") => {
    switch (platform) {
      case "macOS":
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/platform/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".home-mdm").should("exist");
          cy.getAttached(".operating-systems").should("exist");
          // "get" because we expect it not to exist
          cy.get(".home-software").should("not.exist");
          cy.get(".activity-feed").should("not.exist");
          if (tier === "premium") {
            cy.getAttached(".hosts-missing").should("exist");
            cy.getAttached(".hosts-low-space").should("exist");
          } else {
            cy.get(".hosts-missing").should("not.exist");
            cy.get(".hosts-low-space").should("not.exist");
          }
        });
        break;
      case "Windows":
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/platform/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".operating-systems").should("exist");
          // "get" because we expect it not to exist
          cy.get(".home-software").should("not.exist");
          cy.get(".activity-feed").should("not.exist");
          if (tier === "premium") {
            cy.getAttached(".hosts-missing").should("exist");
            cy.getAttached(".hosts-low-space").should("exist");
          } else {
            cy.get(".hosts-missing").should("not.exist");
            cy.get(".hosts-low-space").should("not.exist");
          }
        });
        break;
      case "Linux":
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/platform/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          // "get" because we expect it not to exist
          cy.get(".home-software").should("not.exist");
          cy.get(".activity-feed").should("not.exist");
          if (tier === "premium") {
            cy.getAttached(".hosts-missing").should("exist");
            cy.getAttached(".hosts-low-space").should("exist");
          } else {
            cy.get(".hosts-missing").should("not.exist");
            cy.get(".hosts-low-space").should("not.exist");
          }
        });
        break;
      case "All":
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/platform/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".home-software").should("exist");
          cy.getAttached(".activity-feed").should("exist");
          if (tier === "premium") {
            cy.getAttached(".hosts-missing").should("exist");
            cy.getAttached(".hosts-low-space").should("exist");
          } else {
            cy.get(".hosts-missing").should("not.exist");
            cy.get(".hosts-low-space").should("not.exist");
          }
        });
        break;
      default:
        // no activity feed on team dashboard
        cy.getAttached(".homepage__wrapper").within(() => {
          cy.findByText(/platform/i).should("exist");
          cy.getAttached(".hosts-summary").should("exist");
          cy.getAttached(".home-software").should("exist");
          cy.get(".activity-feed").should("not.exist");
          if (tier === "premium") {
            cy.getAttached(".hosts-missing").should("exist");
            cy.getAttached(".hosts-low-space").should("exist");
          } else {
            cy.get(".hosts-missing").should("not.exist");
            cy.get(".hosts-low-space").should("not.exist");
          }
        });
        break;
    }
  },

  verifiesFilteredHostByPlatform: (platform: string) => {
    if (platform === "none") {
      cy.findByText(/view all hosts/i).click();
      cy.findByRole("status", { name: /hosts filtered by/i }).should(
        "not.exist"
      );
    } else {
      cy.findByText(/view all hosts/i).click();
      cy.findByRole("status", {
        name: `hosts filtered by ${platform}`,
      }).should("exist");
    }
  },
};

export default dashboardPage;

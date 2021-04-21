describe("Fleet Email API", () => {
  it("returns JSON data for test@fleetdm.com", () => {
    cy.request(
      "http://localhost:8025/api/v2/search?kind=to&query=test@fleetdm.com"
    )
      .its("headers")
      .its("content-type")
      .should("include", "application/json");
  });
});

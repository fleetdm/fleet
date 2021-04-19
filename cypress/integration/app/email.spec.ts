import * as path from "path";

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

describe("Hello from Fleet Email API", () => {
  it("Checks last email is Hello from Fleet", () => {
    cy.request("http://localhost:8025/api/v2/messages").then((response) => {
      console.log(response.body);
      expect(response.status).to.eq(200);
      // response.body is an object
      expect(response.body.items[0].To[0]).to.have.property("Domain");
      // to.have.length(500);
      // expect(response).to.have.property("headers");
      // expect(response).to.have.property("duration");
    });
  });
});

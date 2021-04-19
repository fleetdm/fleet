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
  it("if last email is Hello from Fleet", () => {
    cy.request("http://localhost:8025/api/v2/messages").then((response) => {
      console.log(response.body);
      expect(response.status).to.eq(200);
      expect(response.body.items[0].To[0]).to.have.property("Domain");
      expect(response.body.items[0].To[0].Mailbox).to.equal("test");
      expect(response.body.items[0].To[0].Domain).to.equal("fleetdm.com");
      expect(response.body.items[0].Content.Headers.Subject[0]).to.equal(
        "Hello from Fleet"
      );
    });
  });
});

import Fleet from "fleet";

describe("Kolide - API client", () => {
  describe("defaults", () => {
    it("sets the base URL", () => {
      expect(Fleet.baseURL).toEqual("http://localhost:8080/api");
    });
  });
});

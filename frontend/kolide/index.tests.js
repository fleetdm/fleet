import Kolide from "kolide";

describe("Kolide - API client", () => {
  describe("defaults", () => {
    it("sets the base URL", () => {
      expect(Kolide.baseURL).toEqual("http://localhost:8080/api");
    });
  });
});

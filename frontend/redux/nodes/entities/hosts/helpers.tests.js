import { parseEntityFunc } from "./helpers";

describe("reduxConfig - hosts helpers", () => {
  describe("parseEntityFunc", () => {
    it("parses an expected CPU string", () => {
      const host = {
        cpu_brand: "Intel(R) Xeon(R) CPU E5-2420 0 @ 1.90GHz",
        cpu_physical_cores: 2,
      };
      expect(parseEntityFunc(host).cpu_type).toEqual("2 x 1.9 GHz");
    });

    it("parses a host missing clock speed", () => {
      const host = {
        cpu_brand: "Intel(R) Xeon(R) CPU E5-242",
        cpu_physical_cores: 2,
      };
      expect(parseEntityFunc(host).cpu_type).toEqual("2 x Unknown GHz");
    });

    it("parses a host missing CPU brand", () => {
      const host = {
        cpu_physical_cores: 2,
      };
      expect(parseEntityFunc(host).cpu_type).toEqual("2 x Unknown GHz");
    });

    it("parses a host missing CPU cores", () => {
      const host = {
        cpu_brand: "Intel(R) Xeon(R) CPU E5-2420 0 @ 1.90GHz",
      };
      expect(parseEntityFunc(host).cpu_type).toEqual("Unknown x 1.9 GHz");
    });

    it("parses a host missing CPU info entirely", () => {
      const host = {};
      expect(parseEntityFunc(host).cpu_type).toEqual(null);
    });
  });
});

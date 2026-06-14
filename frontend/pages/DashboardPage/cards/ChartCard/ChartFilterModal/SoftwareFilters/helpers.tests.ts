import {
  EPSS_RANGE_HELP,
  getEpssError,
  hasEpssErrors,
  isEpssActive,
  isEpssRangeInvalid,
} from "./helpers";

describe("SoftwareFilters helpers", () => {
  describe("getEpssError", () => {
    it("treats empty input as valid (unset)", () => {
      expect(getEpssError("")).toBeNull();
      expect(getEpssError("   ")).toBeNull();
    });

    it("accepts values within 0–100", () => {
      expect(getEpssError("0")).toBeNull();
      expect(getEpssError("50")).toBeNull();
      expect(getEpssError("100")).toBeNull();
    });

    it("rejects out-of-range and non-numeric values", () => {
      expect(getEpssError("-1")).toBe(EPSS_RANGE_HELP);
      expect(getEpssError("101")).toBe(EPSS_RANGE_HELP);
      expect(getEpssError("abc")).toBe(EPSS_RANGE_HELP);
    });
  });

  describe("isEpssRangeInvalid", () => {
    it("is false unless both bounds are present, valid, and min > max", () => {
      expect(isEpssRangeInvalid("", "")).toBe(false);
      expect(isEpssRangeInvalid("", "5")).toBe(false);
      expect(isEpssRangeInvalid("5", "10")).toBe(false);
      expect(isEpssRangeInvalid("abc", "5")).toBe(false); // per-field error instead
    });

    it("is true when min > max", () => {
      expect(isEpssRangeInvalid("10", "5")).toBe(true);
    });
  });

  describe("hasEpssErrors", () => {
    it("is true for any field error or inverted range", () => {
      expect(hasEpssErrors("-1", "")).toBe(true);
      expect(hasEpssErrors("", "200")).toBe(true);
      expect(hasEpssErrors("10", "5")).toBe(true);
    });

    it("is false for valid/empty input", () => {
      expect(hasEpssErrors("", "")).toBe(false);
      expect(hasEpssErrors("5", "90")).toBe(false);
    });
  });

  describe("isEpssActive", () => {
    it("treats empty or the full 0–100 range as inactive", () => {
      expect(isEpssActive("", "")).toBe(false);
      expect(isEpssActive("0", "100")).toBe(false);
    });

    it("is active when min > 0 or max < 100", () => {
      expect(isEpssActive("1", "100")).toBe(true);
      expect(isEpssActive("0", "99")).toBe(true);
    });
  });
});

import { SEVERITY_SCORE_RANGE_ERROR } from "components/SeverityFilter";

import {
  EPSS_RANGE_HELP,
  EPSS_RANGE_INVALID_MSG,
  getEpssError,
  isEpssActive,
  isEpssRangeInvalid,
  NO_CATEGORIES_MSG,
  validateSoftwareFilters,
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

  describe("validateSoftwareFilters", () => {
    const valid = {
      categories: ["os"],
      epssMin: "",
      epssMax: "",
      minScore: "",
      maxScore: "",
    };

    it("reports nothing when every field is valid or unset", () => {
      expect(validateSoftwareFilters(valid)).toEqual({});
      expect(
        validateSoftwareFilters({
          ...valid,
          categories: ["os", "adobe"],
          epssMin: "5",
          epssMax: "90",
          minScore: "0.1",
          maxScore: "9.9",
        })
      ).toEqual({});
    });

    it("reports an empty category selection", () => {
      expect(validateSoftwareFilters({ ...valid, categories: [] })).toEqual({
        categories: NO_CATEGORIES_MSG,
      });
    });

    it("reports out-of-range EPSS values per field", () => {
      expect(
        validateSoftwareFilters({ ...valid, epssMin: "-1", epssMax: "200" })
      ).toEqual({
        epssMin: EPSS_RANGE_HELP,
        epssMax: EPSS_RANGE_HELP,
      });
    });

    it("attaches an inverted EPSS range to the maximum, the dependent field", () => {
      expect(
        validateSoftwareFilters({ ...valid, epssMin: "10", epssMax: "5" })
      ).toEqual({ epssMax: EPSS_RANGE_INVALID_MSG });
    });

    it("prefers an EPSS field's own format error over the range check", () => {
      // Showing "maximum at or above the minimum" for an unparseable bound
      // would describe the wrong problem.
      expect(
        validateSoftwareFilters({ ...valid, epssMin: "abc", epssMax: "5" })
      ).toEqual({ epssMin: EPSS_RANGE_HELP });
    });

    // The CVSS rules belong to validateSeverityScores. All this module does is
    // rename its two fields, so one case pins that.
    it("renames the severity errors to the Software tab's field names", () => {
      expect(
        validateSoftwareFilters({ ...valid, minScore: "11", maxScore: "5.55" })
      ).toEqual({
        cvssMin: SEVERITY_SCORE_RANGE_ERROR,
        cvssMax: SEVERITY_SCORE_RANGE_ERROR,
      });
    });

    it("reports every invalid field at once, for the submit checkpoint", () => {
      expect(
        validateSoftwareFilters({
          categories: [],
          epssMin: "-1",
          epssMax: "",
          minScore: "11",
          maxScore: "",
        })
      ).toEqual({
        categories: NO_CATEGORIES_MSG,
        epssMin: EPSS_RANGE_HELP,
        cvssMin: SEVERITY_SCORE_RANGE_ERROR,
      });
    });
  });
});

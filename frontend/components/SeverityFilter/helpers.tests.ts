import {
  ANY_SEVERITY_VALUE,
  SEVERITY_RANGE_INVALID_MSG,
  SEVERITY_SCORE_RANGE_ERROR,
  severityFilters,
  severityForRange,
  severityValueLabel,
  SeverityValue,
  validateSeverityScores,
} from "./helpers";

describe("severityForRange", () => {
  it("treats an unset range as Any severity", () => {
    expect(severityForRange(undefined, undefined)).toBe(ANY_SEVERITY_VALUE);
  });

  it("matches a preset when both bounds line up", () => {
    expect(severityForRange(7, 8.9)).toBe("high");
    expect(severityForRange(9, 10)).toBe("critical");
  });

  it("resolves the whole scale back to Any, not to a Custom 0-10", () => {
    expect(severityForRange(0, 10)).toBe(ANY_SEVERITY_VALUE);
  });

  it("falls back to Custom for anything that is not a band", () => {
    expect(severityForRange(7, undefined)).toBe("custom");
    expect(severityForRange(undefined, 0)).toBe("custom");
    expect(severityForRange(0, 6.5)).toBe("custom");
    expect(severityForRange(2.5, 6)).toBe("custom");
  });
});

describe("severityFilters", () => {
  it("is empty when neither bound is entered", () => {
    expect(severityFilters({ minScore: "", maxScore: "" })).toStrictEqual({});
  });

  it("is empty for the whole scale, which narrows nothing", () => {
    expect(severityFilters({ minScore: "0", maxScore: "10" })).toStrictEqual(
      {}
    );
  });

  it("reads a preset's bounds straight off the scores it carries", () => {
    expect(severityFilters({ minScore: "9", maxScore: "10" })).toStrictEqual({
      min: 9,
      max: 10,
    });
  });

  it("omits a bound left empty, leaving that end open", () => {
    expect(severityFilters({ minScore: "2.5", maxScore: "" })).toStrictEqual({
      min: 2.5,
    });
    expect(severityFilters({ minScore: "", maxScore: "6" })).toStrictEqual({
      max: 6,
    });
    expect(severityFilters({ minScore: "", maxScore: "10" })).toStrictEqual({
      max: 10,
    });
  });

  it("keeps an explicit 0 bound", () => {
    expect(severityFilters({ minScore: "0", maxScore: "6" })).toStrictEqual({
      min: 0,
      max: 6,
    });
    expect(severityFilters({ minScore: "0", maxScore: "" })).toStrictEqual({
      min: 0,
    });
  });
});

describe("severityValueLabel", () => {
  const label = (severity: SeverityValue, minScore = "", maxScore = "") =>
    severityValueLabel({ severity, minScore, maxScore });

  it("names a preset and nothing more", () => {
    expect(label("critical")).toBe("Critical");
    expect(label("high")).toBe("High");
    expect(label("medium")).toBe("Medium");
    expect(label("low")).toBe("Low");
    expect(label("any")).toBe("Any");
  });

  it("keeps the entered range for Custom, whose name alone says nothing", () => {
    expect(label("custom", "2.5", "6")).toBe("Custom (2.5 to 6)");
  });

  it("widens a half-open Custom range to the end of the scale", () => {
    expect(label("custom", "2.5", "")).toBe("Custom (2.5 to 10)");
    expect(label("custom", "", "6")).toBe("Custom (0 to 6)");
    expect(label("custom", "0", "0")).toBe("Custom (0 to 0)");
  });

  it("names no range for a Custom selection that narrows nothing", () => {
    expect(label("custom", "", "")).toBe("Custom");
    expect(label("custom", "0", "10")).toBe("Custom");
  });
});

describe("validateSeverityScores", () => {
  it("returns no errors for empty or valid scores", () => {
    expect(validateSeverityScores({ minScore: "", maxScore: "" })).toEqual({});
    expect(
      validateSeverityScores({ minScore: "0.1", maxScore: "9.9" })
    ).toEqual({});
  });

  it("flags out-of-range and over-precise scores per field", () => {
    expect(validateSeverityScores({ minScore: "11", maxScore: "" })).toEqual({
      minScore: SEVERITY_SCORE_RANGE_ERROR,
    });
    expect(validateSeverityScores({ minScore: "", maxScore: "5.55" })).toEqual({
      maxScore: SEVERITY_SCORE_RANGE_ERROR,
    });
  });

  it("attaches an inverted range to the maximum, the dependent field", () => {
    expect(validateSeverityScores({ minScore: "7", maxScore: "3" })).toEqual({
      maxScore: SEVERITY_RANGE_INVALID_MSG,
    });
  });

  it("does not let an inverted range displace a field's own error", () => {
    expect(validateSeverityScores({ minScore: "7", maxScore: "5.55" })).toEqual(
      {
        maxScore: SEVERITY_SCORE_RANGE_ERROR,
      }
    );
    expect(validateSeverityScores({ minScore: "11", maxScore: "3" })).toEqual({
      minScore: SEVERITY_SCORE_RANGE_ERROR,
    });
  });
});

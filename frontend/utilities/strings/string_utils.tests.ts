import stringUtils from "utilities/strings";

describe("parseDuration", () => {
  it("throws if duration empty", () => {
    expect(() => stringUtils.parseDuration("")).toThrow(
      "invalid duration value"
    );
  });

  it("converts duration strings to milliseconds", () => {
    const testCases: { [duration: string]: number } = {
      "1h0m0s": 3_600_000,
      "1h": 3_600_000,
      "30m": 1_800_000,
      "-1h0m0s": -3_600_000,
      "1h30m0s": 5_400_000,
    };

    Object.keys(testCases).forEach((duration) => {
      expect(stringUtils.parseDuration(duration)).toBe(testCases[duration]);
    });
  });
});

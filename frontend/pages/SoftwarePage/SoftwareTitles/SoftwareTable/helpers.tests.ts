import { isValidNumber } from "./helpers";

describe("isValidNumber", () => {
  // Test valid numbers
  it("returns true for valid numbers", () => {
    expect(isValidNumber(0)).toBe(true);
    expect(isValidNumber(42)).toBe(true);
    expect(isValidNumber(-10)).toBe(true);
    expect(isValidNumber(3.14)).toBe(true);
  });

  // Test invalid inputs
  it("returns false for non-number inputs", () => {
    expect(isValidNumber("42")).toBe(false);
    expect(isValidNumber(null)).toBe(false);
    expect(isValidNumber(undefined)).toBe(false);
    expect(isValidNumber({})).toBe(false);
    expect(isValidNumber([])).toBe(false);
    expect(isValidNumber(true)).toBe(false);
  });

  // Test NaN
  it("returns false for NaN", () => {
    expect(isValidNumber(NaN)).toBe(false);
  });

  // Test with min value
  it("respects min value when provided", () => {
    expect(isValidNumber(5, 0)).toBe(true);
    expect(isValidNumber(5, 5)).toBe(true);
    expect(isValidNumber(5, 6)).toBe(false);
  });

  // Test with max value
  it("respects max value when provided", () => {
    expect(isValidNumber(5, undefined, 10)).toBe(true);
    expect(isValidNumber(5, undefined, 5)).toBe(true);
    expect(isValidNumber(5, undefined, 4)).toBe(false);
  });

  // Test with both min and max values
  it("respects both min and max values when provided", () => {
    expect(isValidNumber(5, 0, 10)).toBe(true);
    expect(isValidNumber(0, 0, 10)).toBe(true);
    expect(isValidNumber(10, 0, 10)).toBe(true);
    expect(isValidNumber(-1, 0, 10)).toBe(false);
    expect(isValidNumber(11, 0, 10)).toBe(false);
  });
});

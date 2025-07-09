import { compareVersions } from "./helpers";

describe("compareVersions", () => {
  it("correctly compares patch increments", () => {
    expect(compareVersions("1.0.0", "1.0.1")).toBe(-1);
    expect(compareVersions("1.0.1", "1.0.0")).toBe(1);
    expect(compareVersions("1.0.0", "1.0.0")).toBe(0);
  });

  it("handles pre-release after stable", () => {
    expect(compareVersions("1.0.0", "1.0.0-rc.1")).toBe(1);
    expect(compareVersions("1.0.0-rc.1", "1.0.0")).toBe(-1);
  });

  it("orders pre-release tags correctly", () => {
    expect(compareVersions("1.0.0-alpha", "1.0.0-beta")).toBe(-1);
    expect(compareVersions("1.0.0-beta", "1.0.0-rc")).toBe(-1);
    expect(compareVersions("1.0.0-rc", "1.0.0")).toBe(-1);
    expect(compareVersions("1.0.0-alpha", "1.0.0-rc")).toBe(-1);
  });

  it("compares numeric suffixes after pre-release tags", () => {
    expect(compareVersions("1.0.0-alpha.1", "1.0.0-alpha.2")).toBe(-1);
    expect(compareVersions("1.0.0-rc.1", "1.0.0-rc.2")).toBe(-1);
    expect(compareVersions("1.0.0-rc.4", "1.0.0-rc.3")).toBe(1);
  });

  it("handles alphanumeric suffixes", () => {
    expect(compareVersions("1.0.0a", "1.0.0b")).toBe(-1);
    expect(compareVersions("1.0.0b", "1.0.0a")).toBe(1);
  });

  it("treats shorter and longer versions as equal if trailing zeros", () => {
    expect(compareVersions("1.0", "1.0.0")).toBe(0);
    expect(compareVersions("1.0.0", "1.0")).toBe(0);
    expect(compareVersions("1.0.0", "1.0.0.0")).toBe(0);
  });

  it("compares numeric segments correctly", () => {
    expect(compareVersions("1.0.9", "1.0.10")).toBe(-1);
    expect(compareVersions("1.0.10", "1.0.9")).toBe(1);
  });

  it("handles date-based versioning", () => {
    expect(compareVersions("2023.12.31", "2024.01.01")).toBe(-1);
    expect(compareVersions("2024.01.01", "2023.12.31")).toBe(1);
  });

  it('handles leading "v" in version strings', () => {
    expect(compareVersions("v1.0.0", "v2.0.0")).toBe(-1);
    expect(compareVersions("v2.0.0", "v1.0.0")).toBe(1);
  });

  it("treats build metadata as equal (if supported)", () => {
    expect(compareVersions("1.0.0+20130313144700", "1.0.0")).toBe(0);
  });

  it("is case-insensitive for pre-release tags", () => {
    expect(compareVersions("1.0.0-Alpha", "1.0.0-alpha")).toBe(0);
    expect(compareVersions("1.0.0-BETA", "1.0.0-beta")).toBe(0);
  });

  it("ignores leading zeros in numeric segments", () => {
    expect(compareVersions("1.01.0", "1.1.0")).toBe(0);
    expect(compareVersions("01.1.0", "1.1.0")).toBe(0);
  });
});

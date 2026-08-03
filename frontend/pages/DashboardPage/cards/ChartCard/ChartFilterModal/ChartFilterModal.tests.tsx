import { PLATFORM_OPTIONS } from "./ChartFilterModal";

describe("ChartFilterModal PLATFORM_OPTIONS", () => {
  it("offers mobile platforms (iOS, iPadOS, Android) alongside desktop", () => {
    const values = PLATFORM_OPTIONS.map((o) => o.value);
    expect(values).toEqual([
      "darwin",
      "windows",
      "linux",
      "chrome",
      "ios",
      "ipados",
      "android",
    ]);
  });

  it("labels the mobile platforms for display", () => {
    const labelFor = (value: string) =>
      PLATFORM_OPTIONS.find((o) => o.value === value)?.label;
    expect(labelFor("ios")).toBe("iOS");
    expect(labelFor("ipados")).toBe("iPadOS");
    expect(labelFor("android")).toBe("Android");
  });
});

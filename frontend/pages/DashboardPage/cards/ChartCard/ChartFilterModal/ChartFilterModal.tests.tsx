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
    const byValue = Object.fromEntries(
      PLATFORM_OPTIONS.map((o) => [o.value, o.label])
    );
    expect(byValue.ios).toBe("iOS");
    expect(byValue.ipados).toBe("iPadOS");
    expect(byValue.android).toBe("Android");
  });
});

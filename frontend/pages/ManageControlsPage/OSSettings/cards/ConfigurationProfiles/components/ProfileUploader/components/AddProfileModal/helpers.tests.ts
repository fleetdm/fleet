import generateCustomTargetLabelKey from "./helpers";

describe("generateCustomTargetLabelKey", () => {
  it("returns empty object when target is not Custom", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "All hosts",
        includeMode: "any",
        includeLabels: { foo: true },
        excludeLabels: {},
      })
    ).toEqual({});
  });

  it("returns labelsIncludeAny when include mode is any", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "Custom",
        includeMode: "any",
        includeLabels: { foo: true, bar: true },
        excludeLabels: {},
      })
    ).toEqual({ labelsIncludeAny: ["foo", "bar"] });
  });

  it("returns labelsIncludeAll when include mode is all", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "Custom",
        includeMode: "all",
        includeLabels: { foo: true },
        excludeLabels: {},
      })
    ).toEqual({ labelsIncludeAll: ["foo"] });
  });

  it("returns labelsExcludeAny when exclude labels are selected", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "Custom",
        includeMode: "any",
        includeLabels: {},
        excludeLabels: { bar: true },
      })
    ).toEqual({ labelsExcludeAny: ["bar"] });
  });

  it("returns both include and exclude keys when both have selections", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "Custom",
        includeMode: "all",
        includeLabels: { foo: true },
        excludeLabels: { bar: true },
      })
    ).toEqual({ labelsIncludeAll: ["foo"], labelsExcludeAny: ["bar"] });
  });

  it("omits keys for empty selections", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "Custom",
        includeMode: "all",
        includeLabels: { foo: false },
        excludeLabels: {},
      })
    ).toEqual({});
  });
});

import { listNamesFromSelectedLabels, generateLabelKey } from "./helpers";

describe("listNamesFromSelectedLabels", () => {
  it("returns names of selected labels", () => {
    expect(
      listNamesFromSelectedLabels({ foo: true, bar: false, baz: true })
    ).toEqual(["foo", "baz"]);
  });

  it("returns empty array when nothing is selected", () => {
    expect(listNamesFromSelectedLabels({ foo: false, bar: false })).toEqual([]);
  });

  it("returns empty array for an empty dict", () => {
    expect(listNamesFromSelectedLabels({})).toEqual([]);
  });
});

describe("generateLabelKey", () => {
  it("returns empty object when target is not Custom", () => {
    expect(
      generateLabelKey("All hosts", "any", { foo: true }, "any", {})
    ).toEqual({});
  });

  it("returns labelsIncludeAny when include mode is any", () => {
    expect(
      generateLabelKey("Custom", "any", { foo: true, bar: true }, "any", {})
    ).toEqual({ labelsIncludeAny: ["foo", "bar"] });
  });

  it("returns labelsIncludeAll when include mode is all", () => {
    expect(
      generateLabelKey("Custom", "all", { foo: true }, "any", {})
    ).toEqual({ labelsIncludeAll: ["foo"] });
  });

  it("returns labelsExcludeAny when exclude mode is any", () => {
    expect(
      generateLabelKey("Custom", "any", {}, "any", { bar: true })
    ).toEqual({ labelsExcludeAny: ["bar"] });
  });

  it("returns labelsExcludeAll when exclude mode is all", () => {
    expect(
      generateLabelKey("Custom", "any", {}, "all", { bar: true })
    ).toEqual({ labelsExcludeAll: ["bar"] });
  });

  it("returns both include and exclude keys when both have selections", () => {
    expect(
      generateLabelKey("Custom", "any", { foo: true }, "all", { bar: true })
    ).toEqual({ labelsIncludeAny: ["foo"], labelsExcludeAll: ["bar"] });
  });

  it("omits keys for empty selections", () => {
    expect(
      generateLabelKey("Custom", "all", { foo: false }, "any", {})
    ).toEqual({});
  });
});

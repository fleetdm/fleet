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
    expect(generateLabelKey("All hosts", "any", { foo: true }, {})).toEqual({});
  });

  it("returns labelsIncludeAny when include mode is any", () => {
    expect(
      generateLabelKey("Custom", "any", { foo: true, bar: true }, {})
    ).toEqual({ labelsIncludeAny: ["foo", "bar"] });
  });

  it("returns labelsIncludeAll when include mode is all", () => {
    expect(generateLabelKey("Custom", "all", { foo: true }, {})).toEqual({
      labelsIncludeAll: ["foo"],
    });
  });

  it("returns labelsExcludeAny when exclude labels are selected", () => {
    expect(generateLabelKey("Custom", "any", {}, { bar: true })).toEqual({
      labelsExcludeAny: ["bar"],
    });
  });

  it("returns both include and exclude keys when both have selections", () => {
    expect(
      generateLabelKey("Custom", "any", { foo: true }, { bar: true })
    ).toEqual({ labelsIncludeAny: ["foo"], labelsExcludeAny: ["bar"] });
  });

  it("omits keys for empty selections", () => {
    expect(generateLabelKey("Custom", "all", { foo: false }, {})).toEqual({});
  });
});

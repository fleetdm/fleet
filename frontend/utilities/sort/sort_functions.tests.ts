import sort from "./sort_functions";

describe("versionAsc function", () => {
  it("compares version segments numerically", () => {
    expect(sort.versionAsc("26.6", "26.10")).toEqual(-1);
    expect(sort.versionAsc("26.10", "26.6")).toEqual(1);
    expect(sort.versionAsc("26.6", "26.6")).toEqual(0);
  });

  it("sorts a missing version before any real version", () => {
    expect(sort.versionAsc(null, "26.6")).toEqual(-1);
    expect(sort.versionAsc(undefined, "26.6")).toEqual(-1);
    expect(sort.versionAsc("26.6", null)).toEqual(1);
    expect(sort.versionAsc(null, undefined)).toEqual(0);
  });
});

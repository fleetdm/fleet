import sort from "./sort_functions";

describe("versionAsc function", () => {
  it("delegates to compareVersions for numeric segments", () => {
    expect(sort.versionAsc("26.6", "26.10")).toEqual(-1);
    expect(sort.versionAsc("26.10", "26.6")).toEqual(1);
    expect(sort.versionAsc("26.6", "26.6")).toEqual(0);
  });

  it("coerces non-string values instead of throwing", () => {
    expect(sort.versionAsc((null as unknown) as string, "26.6")).toEqual(-1);
    expect(sort.versionAsc((undefined as unknown) as string, "26.6")).toEqual(
      -1
    );
  });
});

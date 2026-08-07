import { listNamesFromSelectedLabels } from "./labels";

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

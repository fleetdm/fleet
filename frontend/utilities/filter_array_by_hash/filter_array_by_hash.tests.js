import filterArrayByHash from "utilities/filter_array_by_hash";

describe("filterArrayByHash", () => {
  const o1 = { foo: "foo", bar: "bar" };
  const o2 = { foo: "fooz", bar: "barz" };
  const array = [o1, o2];

  it("filters the array to all objects that include the filter strings", () => {
    const filter1 = { foo: "foo", bar: "bar" };
    const filter2 = { foo: "", bar: "bar" };
    const filter3 = { foo: "" };
    const filter4 = { foo: "Fooz", bar: "bar" };
    const filter5 = {};
    const filter6 = { bar: "Foo" };

    expect(filterArrayByHash(array, filter1)).toEqual(array);
    expect(filterArrayByHash(array, filter2)).toEqual(array);
    expect(filterArrayByHash(array, filter3)).toEqual(array);
    expect(filterArrayByHash(array, filter4)).toEqual([o2]);
    expect(filterArrayByHash(array, filter5)).toEqual(array);
    expect(filterArrayByHash(array, filter6)).toEqual([]);
  });
});

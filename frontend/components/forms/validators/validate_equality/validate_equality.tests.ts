import validateEquality from "./index";

describe("validateEquality - validator", () => {
  it("returns true for equal inputs", () => {
    expect(validateEquality("thegnarco", "thegnarco")).toEqual(true);
    expect(validateEquality(1, 1)).toEqual(true);
    expect(validateEquality(1.0, 1)).toEqual(true);
    expect(validateEquality(["thegnarco"], ["thegnarco"])).toEqual(true);
    expect(validateEquality({ hello: "world" }, { hello: "world" })).toEqual(
      true
    );
    expect(
      validateEquality({ foo: { bar: "baz" } }, { foo: { bar: "baz" } })
    ).toEqual(true);
  });

  it("returns false for unequal inputs", () => {
    expect(validateEquality("thegnarco", "thegnar")).toEqual(false);
    expect(validateEquality(1, "thegnar")).toEqual(false);
    expect(validateEquality(["thegnarco"], [1])).toEqual(false);
    expect(validateEquality({ hello: "world" }, { hello: "foo" })).toEqual(
      false
    );
    expect(
      validateEquality({ foo: { bar: "baz" } }, { foo: { bar: "foo" } })
    ).toEqual(false);
  });
});

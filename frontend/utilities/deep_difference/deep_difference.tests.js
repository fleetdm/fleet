import deepDifference from "utilities/deep_difference";

describe("deepDifference - utility", () => {
  it("returns the difference for 2 un-nested objects", () => {
    const obj1 = { id: 1, first_name: "Joe", last_name: "Smith" };
    const obj2 = { id: 1, first_name: "Joe", last_name: "Smyth" };

    expect(deepDifference(obj1, obj2)).toEqual({ last_name: "Smith" });
    expect(deepDifference(obj2, obj1)).toEqual({ last_name: "Smyth" });
  });

  it("returns the difference for 2 nested objects", () => {
    const obj1 = {
      profile: { id: 1, first_name: "Joe", last_name: "Smith" },
      preferences: { email: true, push: false },
      post_ids: [1, 2, 3],
    };
    const obj2 = {
      profile: { id: 1, first_name: "Joe", last_name: "Smyth" },
      preferences: { email: false, push: false },
      post_ids: [1, 3],
    };

    expect(deepDifference(obj1, obj2)).toEqual({
      profile: { last_name: "Smith" },
      preferences: { email: true },
      post_ids: [2],
    });

    expect(deepDifference(obj2, obj1)).toEqual({
      profile: { last_name: "Smyth" },
      preferences: { email: false },
    });
  });

  it("returns the difference for 1 nested object and 1 non-nested object", () => {
    const obj1 = {
      profile: { id: 1, first_name: "Joe", last_name: "Smith" },
      preferences: { email: true, push: false },
      post_ids: [1, 2, 3],
    };
    const obj2 = { profile: "my profile", preferences: "my preferences" };

    expect(deepDifference(obj1, obj2)).toEqual(obj1);
    expect(deepDifference(obj2, obj1)).toEqual(obj2);
  });

  it("returns an empty array when comparing an empty array against a non-empty array", () => {
    const obj1 = { pack_name: "My Pack", label_ids: [] };
    const obj2 = { pack_name: "My Pack", label_ids: [1, 2] };

    expect(deepDifference(obj1, obj2)).toEqual({ label_ids: [] });
  });
});

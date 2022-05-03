import replaceArrayItem from "utilities/replace_array_item";

describe("replaceArrayItem - utility", () => {
  it("substitutes the item in the array with its replacement", () => {
    const groob = { id: 23, name: "Groob" };
    const jason = { id: 78, name: "Jason" };
    const john = { id: 78, name: "John" };
    const zach = { id: 78, name: "Zach" };
    const mike = { id: 78, name: "Mike" };
    const arr1 = [101, 102, 2, 104];
    const arr2 = [groob, jason, john];

    expect(replaceArrayItem(arr1, 2, "hi")).toEqual([101, 102, "hi", 104]);
    expect(replaceArrayItem(arr1, 103, "hi")).toEqual(arr1);
    expect(replaceArrayItem(arr2, jason, zach)).toEqual([groob, zach, john]);
    expect(replaceArrayItem(arr2, mike, zach)).toEqual(arr2);
  });
});

import { getBuiltinPlatformLabelId, ILabelSummary } from "./label";

describe("getBuiltinPlatformLabelId", () => {
  const labels: ILabelSummary[] = [
    { id: 6, name: "macOS", label_type: "builtin" },
    { id: 10, name: "MS Windows", label_type: "builtin" },
    { id: 11, name: "All Linux", label_type: "regular" }, // user-created imposter
  ];

  it("returns the id of the built-in label for a platform", () => {
    expect(getBuiltinPlatformLabelId(labels, "darwin")).toBe(6);
    expect(getBuiltinPlatformLabelId(labels, "windows")).toBe(10);
  });

  it("never matches a user-created label with the same name", () => {
    expect(getBuiltinPlatformLabelId(labels, "linux")).toBeUndefined();
  });

  it("returns undefined when labels haven't loaded", () => {
    expect(getBuiltinPlatformLabelId(undefined, "darwin")).toBeUndefined();
  });
});

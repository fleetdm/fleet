import validatePresence from "./index";

const validInputs = [[1, 2, 3], { hello: "world" }, "hi@thegnar.co"];

const invalidInputs = ["", undefined, false, null];

describe("validatePresence - validator", () => {
  it("returns true for valid inputs", () => {
    validInputs.forEach((input) => {
      expect(validatePresence(input)).toEqual(true);
    });
  });

  it("returns false for invalid inputs", () => {
    invalidInputs.forEach((input) => {
      expect(validatePresence(input)).toEqual(false);
    });
  });
});

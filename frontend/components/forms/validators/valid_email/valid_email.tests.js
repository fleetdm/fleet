import validEmail from "./index";

describe("valid_email - validators", () => {
  const validEmails = ["hi@thegnar.co", "hi@gnar.dog", "kolide@gmail.com"];
  const invalidEmails = ["www.thegnar.co", "bill@shakespeare"];

  it("returns true for valid emails", () => {
    validEmails.forEach((email) => {
      expect(validEmail(email)).toEqual(true);
    });
  });

  it("returns false for invalid emails", () => {
    invalidEmails.forEach((email) => {
      expect(validEmail(email)).toEqual(false);
    });
  });
});

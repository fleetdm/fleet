import validPassword from "components/forms/validators/valid_password";

describe("validPassword", () => {
  it("is invalid if the password is not at least 7 characters including a number and a symbol", () => {
    const tooShort = "abc12!";
    const noSymbols = "abc12456";
    const noLetters = "$%#12456";
    const noNumbers = "password$%#";
    const allLetters = "mypassword";
    const allNumbers = "123456789";
    const allSymbols = "!@#$%^&*()";

    const invalidPasswords = [
      tooShort,
      noSymbols,
      noLetters,
      noNumbers,
      allLetters,
      allNumbers,
      allSymbols,
    ];

    invalidPasswords.map((password) => {
      return expect(validPassword(password)).toEqual(
        false,
        `expected ${password} to not be valid`
      );
    });
  });

  it("is valid if the password is at least 7 characters and includes a number and a symbol", () => {
    const validPasswords = [
      "p@assw0rd",
      "This should be v4lid!",
      "admin123.",
      "pRZ'bW,6'6o}HnpL62",
    ];

    validPasswords.map((password) => {
      return expect(validPassword(password)).toEqual(
        true,
        `expected ${password} to be valid`
      );
    });
  });
});

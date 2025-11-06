import validPassword from "components/forms/validators/valid_password";

describe("validPassword", () => {
  it("is invalid if the password is not at least 12 characters including a number and a symbol", () => {
    const tooShort = "abc12!";
    const noSymbols = "abc12456aaaa";
    const noLetters = "$%#12456!!!!";
    const noNumbers = "password$%#xxx";
    const allLetters = "mypasswordxx";
    const allNumbers = "123456789111";
    const allSymbols = "!@#$%^&*()!!!!";
    const tooLong =
      "asasasasasasasasasasasasasasasasasasasasasasasasasasasasasasasasasasasas1!";

    const invalidPasswords = [
      {
        password: tooShort,
        error: "Password must be at least 12 characters",
        error_code: "too_short",
      },
      {
        password: noSymbols,
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: noLetters,
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: noNumbers,
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: allLetters,
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: allNumbers,
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: allSymbols,
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: tooLong,
        error: "Password is over the character limit",
        error_code: "too_long",
      },
    ];

    invalidPasswords.map((test) => {
      return expect(validPassword(test.password)).toEqual(
        { isValid: false, error: test.error, error_code: test.error_code },
        `expected ${test.password} to not be valid`
      );
    });
  });

  it("is valid if the password is at least 12 characters and includes a number and a symbol", () => {
    const validPasswords = [
      "p@assw0rd123",
      "This should be v4lid!",
      "admin123.pass",
      "pRZ'bW,6'6o}HnpL62",
    ];

    validPasswords.map((password) => {
      return expect(validPassword(password)).toEqual(
        { isValid: true, error: "", error_code: "" },
        `expected ${password} to be valid`
      );
    });
  });
});

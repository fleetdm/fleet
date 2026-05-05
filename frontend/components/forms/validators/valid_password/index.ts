const LETTER_PRESENT = /[a-z]+/i;
const NUMBER_PRESENT = /[0-9]+/;
const SYMBOL_PRESENT = /\W+/;

export default (password = "") => {
  let error = "";
  let error_code = "";
  if (password.length < 12) {
    error = "Password must be at least 12 characters";
    error_code = "too_short";
  } else if (password.length > 48) {
    error = "Password is over the character limit";
    error_code = "too_long";
  } else if (
    !(
      LETTER_PRESENT.test(password) &&
      NUMBER_PRESENT.test(password) &&
      SYMBOL_PRESENT.test(password)
    )
  ) {
    error = "Password must meet the criteria below";
    error_code = "invalid_format";
  }
  return { isValid: !error, error, error_code };
};

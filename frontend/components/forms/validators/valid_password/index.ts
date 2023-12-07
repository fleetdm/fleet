const LETTER_PRESENT = /[a-z]+/i;
const NUMBER_PRESENT = /[0-9]+/;
const SYMBOL_PRESENT = /\W+/;

export default (password = "") => {
  return (
    password.length >= 12 &&
    LETTER_PRESENT.test(password) &&
    NUMBER_PRESENT.test(password) &&
    SYMBOL_PRESENT.test(password)
  );
};

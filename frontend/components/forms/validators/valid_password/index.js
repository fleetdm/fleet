const LETTER_PRESENT = /[a-z]+/i;
const NUMBER_PRESENT = /[0-9]+/;
const SYMBOL_PRESENT = /[!@#\$%\^&\*\(\)]+/i;

const noWhitespace = (password) => {
  return password.indexOf(' ') === -1;
};

export default (password = '') => {
  return password.length >= 7 &&
    noWhitespace(password) &&
    LETTER_PRESENT.test(password) &&
    NUMBER_PRESENT.test(password) &&
    SYMBOL_PRESENT.test(password);
};

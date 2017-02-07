import expect from 'expect';

import validPassword from 'components/forms/validators/valid_password';

describe('validPassword', () => {
  it('is invalid if the password is not at least 7 characters including a number and a symbol', () => {
    const tooShort = 'abc12!';
    const noSymbols = 'abc12456';
    const noLetters = '$%#12456';
    const noNumbers = 'password$%#';
    const allLetters = 'mypassword';
    const allNumbers = '123456789';
    const allSymbols = '!@#$%^&*()';
    const containsSpace = 'p@ ssw0rd';

    const invalidPasswords = [tooShort, noSymbols, noLetters, noNumbers, allLetters, allNumbers, allSymbols, containsSpace];

    invalidPasswords.map((password) => {
      return expect(validPassword(password)).toEqual(false, `expected ${password} to not be valid`);
    });
  });

  it('is valid if the password is at least 7 characters and includes a number and a symbol', () => {
    expect(validPassword('p@ssw0rd')).toEqual(true, 'expected p@ssw0rd to be valid');
  });
});

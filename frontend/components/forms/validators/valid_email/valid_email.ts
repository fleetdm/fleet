// https://github.com/validatorjs/validator.js/blob/master/README.md#validators

import isEmail from "validator/lib/isEmail";

export default (email: string): boolean => {
  return isEmail(email);
};

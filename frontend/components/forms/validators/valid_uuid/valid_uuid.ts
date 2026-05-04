// https://github.com/validatorjs/validator.js/blob/master/README.md#validators
import isUUID from "validator/lib/isUUID";

export default (val: string) => {
  return isUUID(val);
};

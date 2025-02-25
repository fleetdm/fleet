// https://github.com/validatorjs/validator.js/blob/master/README.md#validators

import isURL from "validator/lib/isURL";

interface IValidUrl {
  url: string;
  /**  Validate protocols specified */
  protocols?: ("http" | "https")[];
}

export default ({ url, protocols }: IValidUrl): boolean => {
  return isURL(url, { protocols });
};

// https://github.com/validatorjs/validator.js/blob/master/README.md#validators

import isURL from "validator/lib/isURL";

interface IValidUrl {
  url: string;
  /**  Validate protocols specified */
  protocols?: ("http" | "https")[];
  allowLocalHost?: boolean;
}

export default ({ url, protocols, allowLocalHost = false }: IValidUrl) => {
  return isURL(url, {
    protocols,
    require_protocol: !!protocols?.length,
    require_tld: !allowLocalHost,
  });
};

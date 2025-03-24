// https://github.com/validatorjs/validator.js/blob/master/README.md#validators

import isURL from "validator/lib/isURL";

interface IValidUrl {
  url: string;
  /**  Validate protocols specified */
  protocols?: ("http" | "https")[];
  allowAnyLocalHost?: boolean;
}

export default ({
  url,
  protocols,
  allowAnyLocalHost = false,
}: IValidUrl): boolean => {
  if (allowAnyLocalHost && url.includes("localhost")) return true;
  // this function also has a `require_valid_protocol` option, though as called it seems to already validate
  // that the URL's protocol is one of those specified
  return isURL(url, { protocols, require_protocol: !!protocols?.length });
};

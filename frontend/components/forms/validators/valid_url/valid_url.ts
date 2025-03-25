// https://github.com/validatorjs/validator.js/blob/master/README.md#validators

import isURL from "validator/lib/isURL";

interface IValidUrl {
  url: string;
  /**  Validate protocols specified */
  protocols?: ("http" | "https")[];
  allowLocalHost?: boolean;
}

export default ({ url, protocols, allowLocalHost }: IValidUrl) => {
  // this function also has a `require_valid_protocol` option, though as called it seems to already validate
  // that the URL's protocol is one of those specified
  if (allowLocalHost) {
    if (
      url.startsWith("http://localhost") ||
      url.startsWith("https://localhost")
    ) {
      return true;
    }
  }
  return isURL(url, {
    protocols,
    require_protocol: !!protocols?.length,
  });
};

// https://github.com/validatorjs/validator.js/blob/master/README.md#validators

import isURL, { IsURLOptions } from "validator/lib/isURL";

interface IValidUrl {
  url: string;
  /**  Validate protocols specified */
  protocols?: ("http" | "https" | "file")[];
  allowLocalHost?: boolean;
}

export default ({ url, protocols, allowLocalHost = false }: IValidUrl) => {
  const options: Partial<IsURLOptions> = {
    protocols,
    require_protocol: !!protocols?.length,
    require_tld: !allowLocalHost,
  };

  // add some additional options specific to file protocol URLs
  if (protocols?.includes("file")) {
    options.allow_underscores = true;
    options.allow_protocol_relative_urls = true;
    options.require_host = false;
  }

  return isURL(url, options);
};

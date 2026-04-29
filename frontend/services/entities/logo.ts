/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { getPathWithQueryParams } from "utilities/url";

export type IOrgLogoMode = "light" | "dark" | "all";

export default {
  upload: (file: File, mode: IOrgLogoMode = "all") => {
    const { LOGO } = endpoints;
    const path = getPathWithQueryParams(LOGO, { mode });

    const formData = new FormData();
    formData.append("logo", file);

    return sendRequest("PUT", path, formData);
  },
  delete: (mode: IOrgLogoMode = "all") => {
    const { LOGO } = endpoints;
    const path = getPathWithQueryParams(LOGO, { mode });

    return sendRequest("DELETE", path);
  },
};

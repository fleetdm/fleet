import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export type FleetdPackageLinux = "deb" | "rpm";

export default {
  load: async (
    type: FleetdPackageLinux,
    arch: string,
    desktop: boolean
  ): Promise<any> => {
    const path = `${endpoints.FLEETD_PACKAGE}?${buildQueryStringFromParams({
      type,
      arch,
      desktop,
    })}`;
    return sendRequest("GET", path);
  },
};

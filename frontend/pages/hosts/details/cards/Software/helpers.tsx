import { QueryParams } from "utilities/url";

// available_for_install string > boolean conversion in parseHostSoftwareQueryParams
const getHostSoftwareFilterFromQueryParams = (queryParams: QueryParams) => {
  const { available_for_install } = queryParams;

  return available_for_install ? "installableSoftware" : "allSoftware";
};

export default getHostSoftwareFilterFromQueryParams;

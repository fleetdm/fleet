import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { ISoftware } from "interfaces/software";

interface IGetSoftwareProps {
  page: number;
  perPage: number;
  orderKey: string;
  orderDir: "asc" | "desc";
  query: string;
  vulnerable: boolean;
  teamId: boolean;
}

type ISoftwareParams = Partial<IGetSoftwareProps>;

const DEFAULT_PAGE = 0;
const PER_PAGE = 8;
const ORDER_KEY = "hosts_count";
const ORDER_DIRECTION = "desc";

export default {
  load: ({
    page = DEFAULT_PAGE,
    perPage = PER_PAGE,
    orderKey = ORDER_KEY,
    orderDir = ORDER_DIRECTION,
    query,
    vulnerable,
    teamId,
  }: ISoftwareParams): ISoftware[] => {
    const { SOFTWARE } = endpoints;
    const pagination = `page=${page}&per_page=${perPage}`;
    const sort = `order_key=${orderKey}&order_direction=${orderDir}`;
    const team = teamId ? `team_id=${teamId}` : '';
    const path = `${SOFTWARE}?${pagination}&${sort}&${team}&${query}&${vulnerable}`;

    // return sendRequest("GET", path);
    return [
      {
        hosts_count: 124,
        id: 1,
        name: "Chrome.app",
        version: "2.1.11",
        source: "Application (macOS)",
        generated_cpe: "",
        vulnerabilities: null
      },
      {
        hosts_count: 112,
        id: 2,
        name: "Figma.app",
        version: "2.1.11",
        source: "Application (macOS)",
        generated_cpe: "",
        vulnerabilities: null
      },
      {
        hosts_count: 78,
        id: 3,
        name: "osquery",
        version: "2.1.11",
        source: "rpm_packages",
        generated_cpe: "",
        vulnerabilities: null
      },
      {
        hosts_count: 78,
        id: 4,
        name: "osquery",
        version: "2.1.11",
        source: "rpm_packages",
        generated_cpe: "",
        vulnerabilities: null
      },
    ];
  },
};

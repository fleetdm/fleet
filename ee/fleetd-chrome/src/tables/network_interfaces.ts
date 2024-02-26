import Table from "./Table";

export default class TableNetworkInterfaces extends Table {
  name = "network_interfaces";
  columns = ["mac", "ipv4", "ipv6"];

  async generate() {
    if (!chrome.enterprise) {
      return {
        data: [],
        warnings: [
          {
            column: "mac",
            error_message: "chrome.enterprise API is not available for network details",
          },
        ],
      };
    }

    // @ts-expect-error @types/chrome doesn't yet have the getNetworkDetails Promise API.
    const networkDetails = (await chrome.enterprise.networkingAttributes.getNetworkDetails()) as chrome.enterprise.networkingAttributes.NetworkDetails;
    const ipv4 = networkDetails.ipv4;
    const ipv6 = networkDetails.ipv6;
    const mac = networkDetails.macAddress;

    return {
      data: [
        {
          mac,
          ipv4,
          ipv6,
        },
      ],
    };
  }
}

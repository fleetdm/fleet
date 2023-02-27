import Table from "./Table";

export default class TableNetworkInterfaces extends Table {
  name = "network_interfaces";
  columns = ["mac", "address", "ipv4", "ipv6"];

  async generate() {
    const {
      ipv4,
      ipv6,
      macAddress: mac,
      // @ts-expect-error @types/chrome doesn't yet have the getNetworkDetails Promise API.
    } = (await chrome.enterprise.networkingAttributes.getNetworkDetails()) as chrome.enterprise.networkingAttributes.NetworkDetails;

    return [
      {
        mac,
        ipv4,
        ipv6,
        address: ipv4,
      },
    ];
  }
}

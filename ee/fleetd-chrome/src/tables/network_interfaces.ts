import Table from "./Table";

export default class TableNetworkInterfaces extends Table {
  name = "network_interfaces";
  columns = ["mac", "ipv4", "ipv6"];

  async generate() {
    let ipv4: string, ipv6: string, mac: string;
    try {
      // @ts-expect-error @types/chrome doesn't yet have the getNetworkDetails Promise API.
      const networkDetails = (await chrome.enterprise.networkingAttributes.getNetworkDetails()) as chrome.enterprise.networkingAttributes.NetworkDetails;
      ipv4 = networkDetails.ipv4;
      ipv6 = networkDetails.ipv6;
      mac = networkDetails.macAddress;
    } catch (err) {
      console.warn(`get network details: ${err}`);
    }
    return [
      {
        mac,
        ipv4,
        ipv6,
      },
    ];
  }
}

import Table from "./Table";

export default class TableNetworkInterfaces extends Table {
  name = "network_interfaces";
  columns = ["mac", "address", "ipv4", "ipv6"];

  async generate() {
    let mac = "",
      ipv4 = "",
      ipv6 = "";

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

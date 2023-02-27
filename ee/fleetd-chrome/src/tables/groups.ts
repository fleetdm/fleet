import Table from "./Table";

// TODO remove this table once we've modified the users query for ChromeOS.

export default class TableGroups extends Table {
  name = "groups";
  columns = ["gid"];

  async generate() {
    return [];
  }
}

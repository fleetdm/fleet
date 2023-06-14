import Table from "./Table";

export default class TableDiskInfo extends Table {
  name = "disk_info";
  columns = ["test1", "test2"];

  async generate() {
    return [
      {
        test1: "test1 val",
        test2: "test2 val",
      },
    ];
  }
}

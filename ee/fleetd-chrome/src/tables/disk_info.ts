import Table from "./Table";

export default class TableDiskInfo extends Table {
  name = "disk_info";
  columns = ["capacity", "id", "name", "type"];

  async generate() {
    let rows = [];
    const disks = (await chrome.system.storage.getInfo()) as chrome.system.storage.StorageUnitInfo[];
    for (let d of disks) {
      rows.push({
        capacity: d.capacity,
        id: d.id,
        name: d.name,
        type: d.type,
      });
    }
    return { data: rows };
  }
}

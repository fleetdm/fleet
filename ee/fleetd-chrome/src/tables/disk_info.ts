import Table from "./Table";

export default class TableDiskInfo extends Table {
  name = "disk_info";
  columns = ["capacity", "id", "name", "type"];

  async generate() {
    let capacity, id, name, type;
    try {
      const disks = (await chrome.system.storage.getInfo()) as chrome.system.storage.StorageUnitInfo[];
      let rows = [];
      for (let d of disks) {
        rows.push({
          capacity: d.capacity,
          id: d.id,
          name: d.name,
          type: d.type,
        });
      }
    } catch (err) {
      console.warn(`get disk info: ${err}`);
    }

    return [
      {
        capacity: capacity,
        id: id,
        name: name,
        type: type,
      },
    ];
  }
}

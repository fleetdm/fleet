import Table from "./Table";

export default class TableDiskInfo extends Table {
  name = "disk_info";
  columns = ["capacity", "id", "name", "type"];

  async generate() {
    let capacity, id, name, type;
    try {
      const diskInfo = (await chrome.system.storage.getInfo()) as chrome.system.storage.StorageUnitInfo[];
      capacity = diskInfo[0].capacity;
      id = diskInfo[0].id
      name = diskInfo[0].name
      type = diskInfo[0].type
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

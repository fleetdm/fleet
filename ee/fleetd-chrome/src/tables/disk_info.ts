import Table from "./Table";

export default class TableDiskInfo extends Table {
  name = "disk_info";
  columns = ["capacity", "id", "name", "type"];

  async generate() {
    let rows = [];
    // Try to get internal storage first.
    const estimate = await navigator.storage.estimate();
    if (estimate && estimate.quota) {
        const internalStorageUnit:chrome.system.storage.StorageUnitInfo = {
          id: 'internal',
          name: 'Internal Storage',
          type: 'fixed',
          capacity: estimate.quota, 
        };
        rows.push(internalStorageUnit);
    }
    // Add any removable storage.
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

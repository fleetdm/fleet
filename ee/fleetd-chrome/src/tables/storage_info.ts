import Table from "./Table";

export default class TableStorageInfo extends Table {
  name = "storage_info";

  columns = [
    "id",
    "name",
    "type",
    "capacity",
  ];

  async generate() {
    let results: chrome.system.storage.StorageUnitInfo[] = []

    try {
      results = await chrome.system.storage.getInfo()
      // ATM this is in the Dev chanel, we can do something like this to get the
      // available space once this API mades its way into the Stable channel.
      // for (var storage of storageInfo) {
        // let capacity = await chrome.system.storage.getAvailableCapacity(storage.id)
        // results.push([storage, capacity])
      // }
    } catch (err) {
      console.warn("getting storage info", err);
    }

    return results.map(x => ({
        capacity: x.capacity,
        id: x.id,
        name: x.name,
        type: x.type,
    }))
  }
}

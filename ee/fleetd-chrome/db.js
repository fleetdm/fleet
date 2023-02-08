import TableOSVersion from "./os_version.mjs";
import TableGeolocation from "./geolocation.js";
import TableSystemInfo from "./system_info.js";

export default class VirtualDatabase {
  constructor(sqlite3, db) {
    this.sqlite3 = sqlite3;
    this.db = db;

    VirtualDatabase.register(sqlite3, db, new TableOSVersion(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableGeolocation(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableSystemInfo(sqlite3, db));
  }

  static register(sqlite3, db, table) {
    sqlite3.create_module(db, table.name, table);
  }

  async query(sql) {
    let rows = [];
    await this.sqlite3.exec(this.db, sql, (row, columns) => {
      // map each row to object
      rows.push(
        Object.fromEntries(columns.map((_, i) => [columns[i], row[i]]))
      );
    });
    return rows;
  }
}

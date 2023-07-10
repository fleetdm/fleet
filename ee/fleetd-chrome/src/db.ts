import SQLiteAsyncESMFactory from "wa-sqlite/dist/wa-sqlite-async.mjs";
import * as SQLite from "wa-sqlite";

// Alphabetical order
import Table from "./tables/Table";
import TableChromeExtensions from "./tables/chrome_extensions";
import TableCpuInfo from "./tables/cpu_info";
import TableCpuTime from "./tables/cpu_time";
import TableDiskInfo from "./tables/disk_info";
import TableGeolocation from "./tables/geolocation";
import TableNetworkInterfaces from "./tables/network_interfaces";
import TableOsqueryInfo from "./tables/osquery_info";
import TableOSVersion from "./tables/os_version";
import TablePrivacyPreferences from "./tables/privacy_preferences";
import TableScreenLock from "./tables/screenlock";
import TableSystemInfo from "./tables/system_info";
import TableSystemState from "./tables/system_state";
import TableTemperatureSensors from "./tables/temperature_sensors";
import TableUsers from "./tables/users";

export default class VirtualDatabase {
  sqlite3: SQLiteAPI;
  db: number;

  private constructor(sqlite3: SQLiteAPI, db: number) {
    this.sqlite3 = sqlite3;
    this.db = db;

    // Alphabetical order
    VirtualDatabase.register(
      sqlite3,
      db,
      new TableChromeExtensions(sqlite3, db)
    );
    VirtualDatabase.register(sqlite3, db, new TableCpuInfo(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableCpuTime(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableDiskInfo(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableGeolocation(sqlite3, db));
    VirtualDatabase.register(
      sqlite3,
      db,
      new TableNetworkInterfaces(sqlite3, db)
    );
    VirtualDatabase.register(sqlite3, db, new TableOSVersion(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableOsqueryInfo(sqlite3, db));
    VirtualDatabase.register(
      sqlite3,
      db,
      new TablePrivacyPreferences(sqlite3, db)
    );
    VirtualDatabase.register(sqlite3, db, new TableScreenLock(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableSystemInfo(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableSystemState(sqlite3, db));
    VirtualDatabase.register(
      sqlite3,
      db,
      new TableTemperatureSensors(sqlite3, db)
    );
    VirtualDatabase.register(sqlite3, db, new TableUsers(sqlite3, db));
  }

  public static async init(): Promise<VirtualDatabase> {
    const module = await SQLiteAsyncESMFactory();
    const sqlite3 = SQLite.Factory(module);
    const db = await sqlite3.open_v2(":memory:");
    return new VirtualDatabase(sqlite3, db);
  }

  static register(sqlite3: SQLiteAPI, db: number, table: Table) {
    sqlite3.create_module(db, table.name, table);
  }

  async query(sql: string): Promise<Record<string, string | number>[]> {
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

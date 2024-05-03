import SQLiteAsyncESMFactory from "wa-sqlite/dist/wa-sqlite-async.mjs";
import * as SQLite from "wa-sqlite";

import Table from "./tables/Table";
import TableChromeExtensions from "./tables/chrome_extensions";
import TableDiskInfo from "./tables/disk_info";
import TableGeolocation from "./tables/geolocation";
import TableNetworkInterfaces from "./tables/network_interfaces";
import TableOsqueryInfo from "./tables/osquery_info";
import TableOSVersion from "./tables/os_version";
import TablePrivacyPreferences from "./tables/privacy_preferences";
import TableScreenLock from "./tables/screenlock";
import TableSystemInfo from "./tables/system_info";
import TableSystemState from "./tables/system_state";
import TableUsers from "./tables/users";

interface ChromeWarning {
  column: string;
  error_message: string;
}
interface ChromeResponse {
  data: Record<string, string>[];
  /** Manually add errors in catch response if table requires multiple APIs requests */
  warnings?: ChromeWarning[];
}

export default class VirtualDatabase {
  sqlite3: SQLiteAPI;
  db: number;
  warnings?: ChromeWarning[];

  private constructor(sqlite3: SQLiteAPI, db: number) {
    this.sqlite3 = sqlite3;
    this.db = db;

    VirtualDatabase.register(
      sqlite3,
      db,
      new TableChromeExtensions(sqlite3, db)
    );
    VirtualDatabase.register(sqlite3, db, new TableDiskInfo(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableGeolocation(sqlite3, db));
    VirtualDatabase.register(
      sqlite3,
      db,
      new TableNetworkInterfaces(sqlite3, db)
    );
    VirtualDatabase.register(
      sqlite3,
      db,
      new TablePrivacyPreferences(sqlite3, db)
    );
    VirtualDatabase.register(sqlite3, db, new TableScreenLock(sqlite3, db));
    VirtualDatabase.register(
      sqlite3,
      db,
      new TableSystemInfo(sqlite3, db, this.warnings)
    );
    VirtualDatabase.register(sqlite3, db, new TableSystemState(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableOSVersion(sqlite3, db));
    VirtualDatabase.register(sqlite3, db, new TableOsqueryInfo(sqlite3, db));
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

  async query(sql: string): Promise<ChromeResponse> {
    this.warnings = null; // clear warnings
    let rows = [];
    await this.sqlite3.exec(this.db, sql, (row, columns) => {
      // map each row to object
      rows.push(
        Object.fromEntries(
          columns.map((_, i) => {
            let [colName, val] = [columns[i], row[i]];
            if (typeof val !== "string") {
              if (typeof val === "boolean") {
                val = val === true ? "1" : "0";
              } else if (val && val.toString) {
                val = val.toString();
              } else {
                this.warnings?.push({
                  column: colName,
                  error_message: `Value is not a string and doesn't have a toString method: ${val}`,
                });
                val = null;
              }
            }
            return [colName, val];
          })
        )
      );
    });
    return { data: rows, warnings: this.warnings };
  }
}

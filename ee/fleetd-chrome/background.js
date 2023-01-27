// @ts-ignore
import SQLiteAsyncESMFactory from "./node_modules/wa-sqlite/dist/wa-sqlite-async.mjs";

import * as SQLite from "./node_modules/wa-sqlite/src/sqlite-api.js";

import TableOSVersion from "./os_version.mjs";
import TableGeolocation from "./geolocation.js";

let globalDB;
(async () => {
  const module = await SQLiteAsyncESMFactory();
  const sqlite3 = SQLite.Factory(module);
  const db = await sqlite3.open_v2(":memory:");
  globalDB = db;

  sqlite3.create_module(db, "os_version", new TableOSVersion(sqlite3, db));
  sqlite3.create_module(db, "geolocation", new TableGeolocation(sqlite3, db));

  await sqlite3.exec(db, `SELECT * from geolocation`, (row, columns) => {
    console.log(columns, row);
  });
  await sqlite3.close(db);
})();

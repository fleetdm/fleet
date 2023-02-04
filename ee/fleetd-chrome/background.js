// @ts-ignore
import SQLiteAsyncESMFactory from "./node_modules/wa-sqlite/dist/wa-sqlite-async.mjs";

import * as SQLite from "./node_modules/wa-sqlite/src/sqlite-api.js";

import TableOSVersion from "./os_version.mjs";
import TableGeolocation from "./geolocation.js";

const register = (sqlite3, db, table) => {
  sqlite3.create_module(db, table.name, table);
};

(async () => {
  const module = await SQLiteAsyncESMFactory();
  const sqlite3 = SQLite.Factory(module);
  const db = await sqlite3.open_v2(":memory:");

  register(sqlite3, db, new TableOSVersion(sqlite3, db));
  register(sqlite3, db, new TableGeolocation(sqlite3, db));

  await sqlite3.exec(db, `SELECT * from geolocation`, (row, columns) => {
    console.log(columns, row);
  });
  await sqlite3.exec(db, `SELECT * from os_version`, (row, columns) => {
    console.log(columns, row);
  });
  await sqlite3.close(db);
})();

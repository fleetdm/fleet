import * as SQLite from "./node_modules/wa-sqlite/src/sqlite-api.js";

const generate = async (c) => {
  console.log("generate", c);
  const data = await navigator.userAgentData.getHighEntropyValues([
    "architecture",
    "model",
    "platformVersion",
    "fullVersionList",
  ]);

  console.log("entropy", data);

  c.rows = [
    {
      name: "ChromeOS",
      version: "test",
      platform: "ChromeOS",
      platform_like: "ChromeOS",
      arch: "x86",
    },
  ];
  c.generated = true;
  console.log("end of generate");
};

export default class TableOSVersion {
  cursorStates = new Map();

  columns = ["platform", "platform_like", "version"];
  rows = [
    ["foo1", "bar1", "baz1"],
    ["foo2", "bar2", "baz2"],
  ];

  /**
   * @param {SQLiteAPI} sqlite3
   * @param {number} db
   */
  constructor(sqlite3, db) {
    this.sqlite3 = sqlite3;
    this.db = db;
  }

  /**
   * @param {number} db
   * @param {*} appData Application data passed to `SQLiteAPI.create_module`.
   * @param {Array<string>} argv
   * @param {number} pVTab
   * @param {{ set: function(string): void}} pzString
   * @returns {number|Promise<number>}
   */
  xConnect(db, appData, argv, pVTab, pzString) {
    const sql = `CREATE TABLE os_version (${this.columns.join(",")})`;
    pzString.set(sql);
    return SQLite.SQLITE_OK;
  }

  /**
   * @param {number} pVTab
   * @param {SQLiteModuleIndexInfo} indexInfo
   * @returns {number|Promise<number>}
   */
  xBestIndex(pVTab, indexInfo) {
    return SQLite.SQLITE_OK;
  }

  /**
   * @param {number} pVTab
   * @returns {number|Promise<number>}
   */
  xDisconnect(pVTab) {
    return SQLite.SQLITE_OK;
  }

  /**
   * @param {number} pVTab
   * @returns {number|Promise<number>}
   */
  xDestroy(pVTab) {
    return SQLite.SQLITE_OK;
  }

  /**
   * @param {number} pVTab
   * @param {number} pCursor
   * @returns {number|Promise<number>}
   */
  xOpen(pVTab, pCursor) {
    this.cursorStates.set(pCursor, {});
    return SQLite.SQLITE_OK;
  }

  /**
   * @param {number} pCursor
   * @returns {number|Promise<number>}
   */
  xClose(pCursor) {
    this.cursorStates.delete(pCursor);
    return SQLite.SQLITE_OK;
  }

  /**
   * @param {number} pCursor
   * @param {number} idxNum
   * @param {string?} idxStr
   * @param {Array<number>} values
   * @returns {number|Promise<number>}
   */
  xFilter(pCursor, idxNum, idxStr, values) {
    return this.handleAsync(async () => {
      const cursorState = this.cursorStates.get(pCursor);
      cursorState.index = 0;

      const data = await navigator.userAgentData.getHighEntropyValues([
        "architecture",
        "model",
        "platformVersion",
        "fullVersionList",
      ]);
      console.log(data);
      this.rows = [[data.platform, data.platform, data.platformVersion]];
      console.log(this.rows);

      return SQLite.SQLITE_OK;
    });
  }

  /**
   * @param {number} pCursor
   * @returns {number|Promise<number>}
   */
  xNext(pCursor) {
    // Advance to the next valid row or EOF.
    const cursorState = this.cursorStates.get(pCursor);
    cursorState.index += 1;
    return SQLite.SQLITE_OK;
  }

  /**
   * @param {number} pCursor
   * @returns {number|Promise<number>}
   */
  xEof(pCursor) {
    const cursorState = this.cursorStates.get(pCursor);
    return cursorState.index >= this.rows.length;
  }

  /**
   * @param {number} pCursor
   * @param {number} pContext
   * @param {number} iCol
   * @returns {number|Promise<number>}
   */
  xColumn(pCursor, pContext, iCol) {
    const cursorState = this.cursorStates.get(pCursor);
    const value = this.rows[cursorState.index][iCol];
    this.sqlite3.result(pContext, value);
    return SQLite.SQLITE_OK;
  }

  /**
   * @param {number} pCursor
   * @param {{ set: function(number): void}} pRowid
   * @returns {number|Promise<number>}
   */
  xRowid(pCursor, pRowid) {
    const cursorState = this.cursorStates.get(pCursor);
    pRowid.set(cursorState.index);
    return SQLite.SQLITE_OK;
  }
}

import * as SQLite from "wa-sqlite";

export default class Table {
  name = "table_name";
  columns = ["platform", "platform_like", "version"];

  cursorStates = new Map();

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
    const sql = `CREATE TABLE ${this.name} (${this.columns.join(",")})`;
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
      cursorState.rows = await this.generate(idxNum, idxStr, values);

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
    return cursorState.index >= cursorState.rows.length;
  }

  /**
   * @param {number} pCursor
   * @param {number} pContext
   * @param {number} iCol
   * @returns {number|Promise<number>}
   */
  xColumn(pCursor, pContext, iCol) {
    const cursorState = this.cursorStates.get(pCursor);
    const value = cursorState.rows[cursorState.index][iCol];
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

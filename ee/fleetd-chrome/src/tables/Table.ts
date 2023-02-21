import * as SQLite from "wa-sqlite";

export default abstract class Table implements SQLiteModule {
  sqlite3: SQLiteAPI;
  db: number;
  name: string;
  columns: string[];

  abstract generate(
    idxNum: number,
    idxString: string,
    values: Array<number>
  ): Promise<Record<string, string | number>[]>;

  // injected by wa-sqlite, but missing from SQLiteModule definition
  handleAsync(f: () => Promise<any>): any {}

  cursorStates = new Map();

  constructor(sqlite3: SQLiteAPI, db: number) {
    this.sqlite3 = sqlite3;
    this.db = db;
  }

  xConnect(
    db: number,
    appData: any, // Application data passed to `SQLiteAPI.create_module`.
    argv: Array<string>,
    pVTab: number,
    pzString: { set: (arg0: string) => void }
  ): number | Promise<number> {
    const sql = `CREATE TABLE ${this.name} (${this.columns.join(",")})`;
    pzString.set(sql);
    return SQLite.SQLITE_OK;
  }

  xBestIndex(
    pVTab: number,
    indexInfo: SQLiteModuleIndexInfo
  ): number | Promise<number> {
    return SQLite.SQLITE_OK;
  }

  xDisconnect(pVTab: number): number | Promise<number> {
    return SQLite.SQLITE_OK;
  }

  xDestroy(pVTab: number): number | Promise<number> {
    return SQLite.SQLITE_OK;
  }

  xOpen(pVTab: number, pCursor: number): number | Promise<number> {
    this.cursorStates.set(pCursor, {});
    return SQLite.SQLITE_OK;
  }

  xClose(pCursor: number): number | Promise<number> {
    this.cursorStates.delete(pCursor);
    return SQLite.SQLITE_OK;
  }

  xFilter(
    pCursor: number,
    idxNum: number,
    idxStr: string | null,
    values: Array<number>
  ): Promise<number> {
    return this.handleAsync(async () => {
      const cursorState = this.cursorStates.get(pCursor);
      cursorState.index = 0;
      cursorState.rows = await this.generate(idxNum, idxStr, values);

      return SQLite.SQLITE_OK;
    });
  }

  xNext(pCursor: number): number | Promise<number> {
    // Advance to the next valid row or EOF.
    const cursorState = this.cursorStates.get(pCursor);
    cursorState.index += 1;
    return SQLite.SQLITE_OK;
  }

  xEof(pCursor: number): number | Promise<number> {
    const cursorState = this.cursorStates.get(pCursor);
    return Number(cursorState.index >= cursorState.rows.length);
  }

  xColumn(
    pCursor: number,
    pContext: number,
    iCol: number
  ): number | Promise<number> {
    // Get the generated rows for this cursor.
    const cursorState = this.cursorStates.get(pCursor);
    // Get the current row.
    const row = cursorState.rows[cursorState.index];
    // Get the column for the row, looking up the column index by the column name.
    const value = row[this.columns[iCol]];

    this.sqlite3.result(pContext, value);
    return SQLite.SQLITE_OK;
  }

  xRowid(
    pCursor: number,
    pRowid: { set: (arg0: number) => void }
  ): number | Promise<number> {
    const cursorState = this.cursorStates.get(pCursor);
    pRowid.set(cursorState.index);
    return SQLite.SQLITE_OK;
  }
}

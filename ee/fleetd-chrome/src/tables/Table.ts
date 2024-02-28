// This is the foundation for all of the tables. We take the low-level SQLite functions and map them
// to an abstraction so that table implementations only need to define their name, columns, and
// generate() function.

import * as SQLite from "wa-sqlite";

/** Creates a single UI friendly string out of chrome tables that return multiple warnings */
const CONCAT_CHROME_WARNINGS = (warnings: ChromeWarning[]): string => {
  const warningStrings = warnings.map(
    (warning) => `Column: ${warning.column} - ${warning.error_message}`
  );
  return warningStrings.join("\n");
};
class cursorState {
  rowIndex: number;
  rows: Record<string, string>[];
}

interface ChromeWarning {
  column: string;
  error_message: string;
}
interface ChromeResponse {
  data: Record<string, string>[];
  /** Manually add errors in catch response if table requires requests to multiple APIs */
  warnings?: ChromeWarning[];
}

export default abstract class Table implements SQLiteModule {
  sqlite3: SQLiteAPI;
  db: number;
  name: string;
  columns: string[];
  cursorStates: Map<number, cursorState>;
  warnings?: ChromeWarning[];

  abstract generate(
    idxNum: number,
    idxString: string,
    values: Array<number>
  ): Promise<ChromeResponse>;

  constructor(sqlite3: SQLiteAPI, db: number, warnings?: ChromeWarning[]) {
    this.sqlite3 = sqlite3;
    this.db = db;
    this.cursorStates = new Map();
    this.warnings = warnings;
  }

  // This is replaced by wa-sqlite when SQLite is loaded up, but missing from the SQLiteModule
  // definition. We add it here to make Typescript happy.
  handleAsync(_: () => Promise<number>): number {
    throw new Error("should be replaced in build");
  }

  // All the methods below are documented in https://www.sqlite.org/vtab.html#virtual_table_methods.

  xConnect(
    db: number,
    appData: any, // Application data passed to `SQLiteAPI.create_module`.
    argv: Array<string>,
    pVTab: number,
    pzErr: DataView,
  ): number {
    // Register the table schema.
    const sql = `CREATE TABLE ${this.name} (${this.columns.join(",")})`;
    this.sqlite3.declare_vtab(db, sql);
    return SQLite.SQLITE_OK;
  }

  xBestIndex(
    pVTab: number,
    indexInfo: SQLiteModuleIndexInfo
  ): number {
    // In the future we might be able to use this for some tables to optimize queries.
    return SQLite.SQLITE_OK;
  }

  xDisconnect(pVTab: number): number {
    return SQLite.SQLITE_OK;
  }

  xDestroy(pVTab: number): number {
    return SQLite.SQLITE_OK;
  }

  xOpen(pVTab: number, pCursor: number): number {
    // Initialize a new cursor state (called at the beginning of a query to the table).
    this.cursorStates.set(pCursor, new cursorState());
    return SQLite.SQLITE_OK;
  }

  xClose(pCursor: number): number {
    // Clean up the cursor state (called when the query completes). Important that we do this so
    // that the resources don't remain allocated after the query completes!
    this.cursorStates.delete(pCursor);
    return SQLite.SQLITE_OK;
  }

  xFilter(
    pCursor: number,
    idxNum: number,
    idxStr: string | null,
    values: Array<number>
  ): number {
    // Generate the actual query results here during this filter call. Store them in the cursor state
    // so that SQLite can request each row and column.
    return this.handleAsync(async () => {
      const cursorState = this.cursorStates.get(pCursor);
      cursorState.rowIndex = 0;
      try {
        const tableDataReturned = await this.generate(idxNum, idxStr, values);

        // Set warnings to this.warnings for database to surface in UI
        if (tableDataReturned.warnings) {
          globalThis.DB.warnings = []; // Reset warnings
          globalThis.DB.warnings = CONCAT_CHROME_WARNINGS(
            tableDataReturned.warnings
          );
        }
        cursorState.rows = tableDataReturned.data;
      } catch (err) {
        // We cannot throw inside SQLITE function because it may cause the wasm stack to run out of memory.
        // See: https://github.com/rhashimoto/wa-sqlite/issues/156#issuecomment-1942477704
        console.warn("Error generating table data: %s", err);
        return SQLite.SQLITE_ERROR;
      }
      return SQLite.SQLITE_OK;
    });
  }

  xNext(pCursor: number): number {
    // Advance the row index for the cursor.
    const cursorState = this.cursorStates.get(pCursor);
    if (!cursorState || !cursorState.rows) {
      return SQLite.SQLITE_ERROR;
    }
    cursorState.rowIndex += 1;
    return SQLite.SQLITE_OK;
  }

  xEof(pCursor: number): number {
    // Check whether we've returned all rows (cursor index is beyond number of rows).
    const cursorState = this.cursorStates.get(pCursor);
    if (!cursorState || !cursorState.rows) {
      return 1;
    }
    return Number(cursorState.rowIndex >= cursorState.rows.length);
  }

  xColumn(
    pCursor: number,
    pContext: number,
    iCol: number
  ): number {
    // Get the generated rows for this cursor.
    const cursorState = this.cursorStates.get(pCursor);
    // Get the current row.
    const row = cursorState.rows[cursorState.rowIndex];
    // Get the column for the row, looking up the column index by the column name.
    const value = row[this.columns[iCol]];
    // Provide the result through calling the sqlite3.result() function, then return a success
    // code.
    this.sqlite3.result(pContext, value);
    return SQLite.SQLITE_OK;
  }

  xRowid(
    pCursor: number,
    pRowid: DataView,
  ): number {
    // Get the current row index.
    const cursorState = this.cursorStates.get(pCursor);
    pRowid.setBigInt64(0, BigInt(cursorState.rowIndex));
    return SQLite.SQLITE_OK;
  }
}

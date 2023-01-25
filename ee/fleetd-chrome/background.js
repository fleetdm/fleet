import { default as sqlite3InitModule } from "./sqlite3.mjs";

let sqlite3;

(async () => {
  sqlite3 = await sqlite3InitModule();

  const capi = sqlite3.capi; /* C-style API */
  const oo = sqlite3.oo1; /* high-level OO API */
  const wasm = sqlite3.wasm;

  console.log(
    "sqlite3 version",
    capi.sqlite3_libversion(),
    capi.sqlite3_sourceid()
  );
  const db = new oo.DB();
  const resultRows = [];
  db.exec({
    sql: "select 1 as foo",
    rowMode: "object",
    resultRows,
  });
  console.log("Result rows:", resultRows);

  const VT = sqlite3.vtab;
  const tmplCols = Object.assign(Object.create(null), {
    A: 0,
    B: 1,
  });

  const vtabTrace = (methodName, ...args) =>
    console.debug(`sqlite3_module::${methodName}():`, ...args);
  const modConfig = {
    /* catchExceptions changes how the methods are wrapped */
    catchExceptions: true,
    name: "vtab2test",
    methods: {
      xCreate(pDb, pAux, argc, argv, ppVtab, pzErr) {
        vtabTrace("xCreate", ...arguments);
        const args = wasm.cArgvToJs(argc, argv);
        vtabTrace("xCreate", "argv:", args);
        const rc = capi.sqlite3_declare_vtab(pDb, "CREATE TABLE ignored(a,b)");
        if (rc === 0) {
          const t = VT.xVtab.create(ppVtab);
          vtabTrace("xCreate", ...arguments, " ppVtab =", t.pointer);
        }
        return rc;
      },
      xConnect: true,
      xDestroy(pVtab) {
        vtabTrace("xDestroy/xDisconnect", pVtab);
        VT.xVtab.dispose(pVtab);
      },
      xDisconnect: true,
      xOpen(pVtab, ppCursor) {
        const t = VT.xVtab.get(pVtab);
        const c = VT.xCursor.create(ppCursor);
        vtabTrace("xOpen", ...arguments, " cursor =", c.pointer);
        c._rowId = 0;
      },
      xClose(pCursor) {
        vtabTrace("xClose", ...arguments);
        const c = VT.xCursor.unget(pCursor);
        c.dispose();
      },
      xNext(pCursor) {
        vtabTrace("xNext", ...arguments);
        const c = VT.xCursor.get(pCursor);
        ++c._rowId;
      },
      xColumn(pCursor, pCtx, iCol) {
        vtabTrace("xColumn", ...arguments);
        const c = VT.xCursor.get(pCursor);
        switch (iCol) {
          case tmplCols.A:
            capi.sqlite3_result_int(pCtx, 1000 + c._rowId);
            break;
          case tmplCols.B:
            capi.sqlite3_result_int(pCtx, 2000 + c._rowId);
            break;
          default:
            sqlite3.SQLite3Error.toss("Invalid column id", iCol);
        }
      },
      xRowid(pCursor, ppRowid64) {
        vtabTrace("xRowid", ...arguments);
        const c = VT.xCursor.get(pCursor);
        VT.xRowid(ppRowid64, c._rowId);
      },
      xEof(pCursor) {
        vtabTrace("xEof", ...arguments);
        return VT.xCursor.get(pCursor)._rowId >= 10;
      },
      xFilter(pCursor, idxNum, idxCStr, argc, argv /* [sqlite3_value* ...] */) {
        vtabTrace("xFilter", ...arguments);
        const c = VT.xCursor.get(pCursor);
        c._rowId = 0;
        const list = capi.sqlite3_values_to_js(argc, argv);
      },
      xBestIndex(pVtab, pIdxInfo) {
        vtabTrace("xBestIndex", ...arguments);
        // const t = VT.xVtab.get(pVtab);
        const pii = VT.xIndexInfo(pIdxInfo);
        pii.$estimatedRows = 10;
        pii.$estimatedCost = 10.0;
        pii.dispose();
      },
    } /* methods */,
  };
  const tmplMod = VT.setupModule(modConfig);
  // db.onclose.disposeAfter.push(tmplMod);
  db.checkRc(
    capi.sqlite3_create_module(db.pointer, modConfig.name, tmplMod.pointer, 0)
  );
  db.exec([
    "create virtual table testvtab2 using ",
    modConfig.name,
    "(arg1 blah, arg2 bloop)",
  ]);
  if (0) {
    /* If we DROP TABLE then xDestroy() is called. If the
             vtab is instead destroyed when the db is closed,
             xDisconnect() is called. */
    db.onclose.disposeBefore.push(function (db) {
      console.debug(
        "Explicitly dropping testvtab2 via disposeBefore handler..."
      );
      db.exec(
        /** DROP TABLE is the only way to get xDestroy() to be called. */
        "DROP TABLE testvtab2"
      );
    });
  }
  const list = db.selectArrays(
    "SELECT a,b FROM testvtab2 where a<9999 and b>1 order by a, b"
    /* Query is shaped so that it will ensure that some
             constraints end up in xBestIndex(). */
  );
  console.log(list);
})();

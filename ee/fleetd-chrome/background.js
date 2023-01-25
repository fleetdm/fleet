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

  const VT = sqlite3.vtab;
  const columns = {
    name: 0,
    version: 1,
    platform: 2,
    platform_like: 3,
    arch: 4,
  };

  const vtabTrace = (methodName, ...args) =>
    console.debug(`sqlite3_module::${methodName}():`, ...args);
  const modConfig = {
    /* catchExceptions changes how the methods are wrapped */
    catchExceptions: true,
    name: "os_version",
    methods: {
      xConnect(pDb, pAux, argc, argv, ppVtab, pzErr) {
        vtabTrace("xCreate", ...arguments);
        const args = wasm.cArgvToJs(argc, argv);
        vtabTrace("xCreate", "argv:", args);
        const colNames = Object.keys(columns).join(", ");
        vtabTrace("xCreate", "colNames:", colNames);
        const rc = capi.sqlite3_declare_vtab(
          pDb,
          `CREATE TABLE os_version(${colNames})`
        );
        if (rc === 0) {
          const t = VT.xVtab.create(ppVtab);
          vtabTrace("xCreate", ...arguments, " ppVtab =", t.pointer);
        }
        return rc;
      },
      // Make this "eponymous-only" meaning it comes attached and can't be attached through a CREATE
      // TABLE statement.
      // https://sqlite.org/vtab.html#eponymous_only_virtual_tables
      xCreate: null,
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
        console.log("rows", c.rows);
        c._rowId += 1;
      },
      xColumn(pCursor, pCtx, iCol) {
        vtabTrace("xColumn", ...arguments);
        const c = VT.xCursor.get(pCursor);
        switch (iCol) {
          case columns.name:
            capi.sqlite3_result_js(pCtx, "name");
            break;
          case columns.version:
            capi.sqlite3_result_js(pCtx, "version");
            break;
          case columns.platform:
            capi.sqlite3_result_js(pCtx, "platform");
            break;
          case columns.platform_like:
            capi.sqlite3_result_js(pCtx, "platform_like");
            break;
          case columns.arch:
            capi.sqlite3_result_js(pCtx, "arch");
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
        return VT.xCursor.get(pCursor)._rowId > 0;
      },
      xFilter(pCursor, idxNum, idxCStr, argc, argv /* [sqlite3_value* ...] */) {
        vtabTrace("xFilter", ...arguments);
        const c = VT.xCursor.get(pCursor);
        c._rowId = 0;
        c.rows = [{ foo: "bar" }, { baz: "bing" }];
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
  const rows = db.selectObjects(
    "SELECT * FROM os_version"
    /* Query is shaped so that it will ensure that some
             constraints end up in xBestIndex(). */
  );
  console.log(rows);
})();

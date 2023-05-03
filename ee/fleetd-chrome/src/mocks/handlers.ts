import { rest } from "msw";
import { readFileSync } from "fs";

import { resolve as _resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

export const handlers = [
  // Return the actual webassembly file
  rest.get(/\/wa-sqlite-async.wasm$/, (_req, res, ctx) => {
    const wasm = readFileSync(
      __dirname + "/../../node_modules/wa-sqlite/dist/wa-sqlite-async.wasm"
    );
    return res(
      ctx.status(200),
      ctx.set("Content-Type", "application/wasm"),
      ctx.body(wasm)
    );
  }),
];

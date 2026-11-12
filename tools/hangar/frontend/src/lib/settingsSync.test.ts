// Run with `node --test`. @types/node isn't a dependency, so tsc sees these as untyped.
// @ts-ignore
import assert from "node:assert/strict";
// @ts-ignore
import { test } from "node:test";
import { createSettingsSync } from "./settingsSync.ts";

function deferred<T>() {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((r) => (resolve = r));
  return { promise, resolve };
}

test("reloads resolving in reverse order apply only the latest", async () => {
  const loads = [deferred<string>(), deferred<string>()];
  let i = 0;
  const sync = createSettingsSync(() => loads[i++].promise);
  const applied: string[] = [];
  sync.setApply((s) => applied.push(s));

  const first = sync.reload();
  const second = sync.reload();
  loads[1].resolve("new");
  loads[0].resolve("old");
  await Promise.all([first, second]);

  assert.deepEqual(applied, ["new"]);
});

const loaders: Record<string, (v: () => string) => () => Promise<string>> = {
  immediate: (v) => () => Promise.resolve(v()),
  deferred: (v) => () => new Promise((r) => setTimeout(() => r(v()), 1)),
};

for (const [name, mk] of Object.entries(loaders)) {
  test(`afterSave's own reload lands after the caller's onChange (${name} loader)`, async () => {
    let disk = "disk";
    const sync = createSettingsSync(mk(() => disk));
    let ui = "";
    sync.setApply((s) => (ui = s));

    await sync.afterSave(async () => {
      disk = "kept-by-store";
    });
    ui = "next"; // the caller's continuation
    await new Promise((r) => setTimeout(r, 20));

    assert.equal(ui, "kept-by-store");
  });
}

test("afterSave reloads after a failed save and rethrows", async () => {
  const sync = createSettingsSync(async () => "disk");
  let ui = "";
  sync.setApply((s) => (ui = s));

  await assert.rejects(
    sync.afterSave(async () => {
      throw new Error("held");
    }),
    /held/,
  );
  await sync.reload();
  assert.equal(ui, "disk");
});

test("a settings:changed reload racing a save reload applies the later one", async () => {
  const loads = [deferred<string>(), deferred<string>()];
  let i = 0;
  const sync = createSettingsSync(() => loads[i++].promise);
  const applied: string[] = [];
  sync.setApply((s) => applied.push(s));

  const event = sync.reload();
  const save = sync.afterSave(async () => {});
  await save;
  loads[0].resolve("event-read");
  loads[1].resolve("save-read");
  await event;
  await new Promise((r) => setTimeout(r, 0));

  assert.deepEqual(applied, ["save-read"]);
});

test("a reload resolving while a later save is in flight is not applied", async () => {
  const loads = [deferred<string>(), deferred<string>()];
  let i = 0;
  const sync = createSettingsSync(() => loads[i++].promise);
  const applied: string[] = [];
  sync.setApply((s) => applied.push(s));

  await sync.afterSave(async () => {}); // save 1 done, reload 1 pending
  const save2 = deferred<void>();
  const second = sync.afterSave(() => save2.promise);
  loads[0].resolve("after-save-1");
  await new Promise((r) => setTimeout(r, 0));
  assert.deepEqual(applied, []);

  save2.resolve();
  await second;
  loads[1].resolve("after-save-2");
  await new Promise((r) => setTimeout(r, 0));
  assert.deepEqual(applied, ["after-save-2"]);
});

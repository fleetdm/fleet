import VirtualDatabase from "./db";

test("Simple query", async () => {
  const db = await VirtualDatabase.init();
  const res = await db.query("select 1");
  expect(res).toEqual([{ "1": 1 }]);
});

import VirtualDatabase from "../db";

describe("screen_lock", () => {
  test("success", async () => {
    chrome.idle.getAutoLockDelay = jest.fn(() =>
      Promise.resolve({ delay: 600 })
    );

    const db = await VirtualDatabase.init();
    const res = await db.query("select * from screen_lock");
    expect(res).toEqual([
      {
        delay: 600,
      },
    ]);
  });
});

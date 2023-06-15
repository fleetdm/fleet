import VirtualDatabase from "../db";

describe("system_state", () => {
  const delay = 600; // Returned from chrome.idle.getAutoLockDelay tested in screen_lock

  test("success active state", async () => {
    // @ts-ignore
    chrome.idle.queryState(delay) = jest.fn(() =>
      Promise.resolve({ idle_state: "active" })
    );

    const db = await VirtualDatabase.init();
    const res = await db.query("select * from system_state");
    expect(res).toEqual([
      {
        idle_state: "active",
      },
    ]);
  });

  // TODO test via Fleet app as you can't test idle from an active Chromebook
  test("success idle state", async () => {
    // @ts-ignore
    chrome.idle.queryState(delay) = jest.fn(() =>
      Promise.resolve({ idle_state: "idle" })
    );

    const db = await VirtualDatabase.init();
    const res = await db.query("select * from system_state");
    expect(res).toEqual([
      {
        idle_state: "idle",
      },
    ]);
  });

  test("success locked state", async () => {
    // @ts-ignore
    chrome.idle.queryState(delay) = jest.fn(() =>
      Promise.resolve({ idle_state: "locked" })
    );

    const db = await VirtualDatabase.init();
    const res = await db.query("select * from system_state");
    expect(res).toEqual([
      {
        idle_state: "locked",
      },
    ]);
  });
});

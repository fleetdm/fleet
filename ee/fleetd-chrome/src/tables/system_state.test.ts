import VirtualDatabase from "../db";

describe("system_state", () => {
  test("success active state", async () => {
    const delay = 600; // Returned from chrome.idle.getAutoLockDelay tested in screen_lock

    // @ts-ignore
    chrome.idle.queryState(delay) = jest.fn((idle_state) =>
      Promise.resolve({ idle_state })
    );

    const db = await VirtualDatabase.init();
    const res = await db.query("select * from system_state");
    expect(res).toEqual([
      {
        idle_state: "active",
      },
    ]);
  });

  test("success idle state", async () => {
    // TODO
    const delay = 600; // Returned from chrome.idle.getAutoLockDelay tested in screen_lock

    // @ts-ignore
    chrome.idle.queryState(delay) = jest.fn((idle_state) =>
      Promise.resolve({ idle_state })
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
    // TODO
    const delay = 600; // Returned from chrome.idle.getAutoLockDelay tested in screen_lock

    // @ts-ignore
    chrome.idle.queryState(delay) = jest.fn((idle_state) =>
      Promise.resolve({ idle_state })
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

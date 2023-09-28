import VirtualDatabase from "../db";

describe("screenlock", () => {
  test("success", async () => {
    chrome.idle.getAutoLockDelay = jest.fn(() => Promise.resolve(600));
    chrome.idle.queryState = jest.fn(() => Promise.resolve("active"));

    const db = await VirtualDatabase.init();

    const res = await db.query("select * from system_state");
    expect(res).toEqual({
      data: [
        {
          idle_state: "active",
        },
      ],
    });
  });
});

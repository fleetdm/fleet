import VirtualDatabase from "../db";

describe("screenlock", () => {
  test("success", async () => {
    chrome.idle.getAutoLockDelay = jest.fn((callback) => callback(600));
    chrome.idle.queryState = jest.fn((_, callback) => callback("active"));

    const db = await VirtualDatabase.init();

    const res = await db.query("select * from system_state");
    expect(res).toEqual({
      data: [
        {
          idle_state: "active",
        },
      ],
      warnings: null,
    });
  });
});

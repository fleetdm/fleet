import VirtualDatabase from "../db";

describe("screenlock", () => {
  test("success", async () => {
    chrome.idle.getAutoLockDelay = jest.fn((callback) => callback(600));

    const db = await VirtualDatabase.init();

    const res = await db.query("select * from screenlock");
    expect(res).toEqual({
      data: [
        {
          enabled: "1",
          grace_period: "600",
        },
      ],
      warnings: null,
    });
  });
});

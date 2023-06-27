import VirtualDatabase from "../db";
import chrome from "jest-chrome";

describe("screenlock", () => {
  test("success", async () => {
    // chrome.idle.getAutoLockDelay = jest.fn(() => Promise.resolve(600));

    // lkjlkj @ts-expect-error Typescript doesn't include the chrome API yet.
    // chrome.idle = {
    //   getAutoLockDelay: jest.fn(() => Promise.resolve({ delay: 600 })),
    // };

    // @ts-ignore
    console.log("chrome.idle.getAutoLockDelay", chrome.idle.getAutoLockDelay);

    const db = await VirtualDatabase.init();
    const res = await db.query("select * from screenlock");
    expect(res).toEqual([
      {
        delay: 600,
      },
    ]);
  }, 30000);
});

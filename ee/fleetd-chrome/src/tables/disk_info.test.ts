import VirtualDatabase from "../db";

const DISK_INFO_MOCK = [
  {
    capacity: "1234",
    id: "123",
    name: "Cell phone (internal storage",
    type: "Removable",
  },
  {
    capacity: "0",
    id: "12",
    name: "Thumbdrive",
    type: "Removable",
  },
];

describe("disk_info", () => {
  test("success", async () => {
    // @ts-ignore
    chrome.system.storage.getInfo = jest.fn(() =>
      Promise.resolve(DISK_INFO_MOCK)
    );

    const db = await VirtualDatabase.init();

    const res = await db.query("select * from disk_info");
    expect(res).toEqual({"data":DISK_INFO_MOCK, "warnings": null});
  });
});
